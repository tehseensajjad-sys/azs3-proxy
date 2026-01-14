package azureblob

import (
	"context"
	"io"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
)

type chunk struct {
	id   int
	data []byte
	err  error
}

type concurrentReader struct {
	client      *blockblob.Client
	ctx         context.Context
	cancel      context.CancelFunc
	blobSize    int64
	chunkSize   int64
	concurrency int
	r           *io.PipeReader
	w           *io.PipeWriter
}

func newConcurrentReader(ctx context.Context, client *blockblob.Client, size int64, concurrency int, chunkSize int64) io.ReadCloser {
	if concurrency <= 0 {
		concurrency = 16
	}
	if chunkSize <= 0 {
		chunkSize = 8 * 1024 * 1024 // 8MB
	}

	pr, pw := io.Pipe()
	cCtx, cancel := context.WithCancel(ctx)

	cr := &concurrentReader{
		client:      client,
		ctx:         cCtx,
		cancel:      cancel,
		blobSize:    size,
		chunkSize:   chunkSize,
		concurrency: concurrency,
		r:           pr,
		w:           pw,
	}

	go cr.run()

	return cr
}

func (cr *concurrentReader) Read(p []byte) (n int, err error) {
	return cr.r.Read(p)
}

func (cr *concurrentReader) Close() error {
	cr.cancel()
	_ = cr.r.Close()
	return nil
}

func (cr *concurrentReader) run() {
	defer cr.w.Close()

	if cr.blobSize == 0 {
		return
	}

	numChunks := int((cr.blobSize + cr.chunkSize - 1) / cr.chunkSize)

	// Create channels
	// jobs: worker input
	jobs := make(chan int)
	// results: worker output
	results := make(chan *chunk, cr.concurrency)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < cr.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for chunkID := range jobs {
				// Check for cancellation
				select {
				case <-cr.ctx.Done():
					return
				default:
				}

				// Download chunk
				start := int64(chunkID) * cr.chunkSize
				count := cr.chunkSize
				if start+count > cr.blobSize {
					count = cr.blobSize - start
				}

				// Pre-allocate buffer for the chunk
				data := make([]byte, count)

				opts := &azblob.DownloadStreamOptions{
					Range: azblob.HTTPRange{
						Offset: start,
						Count:  count,
					},
				}

				resp, err := cr.client.DownloadStream(cr.ctx, opts)
				var readErr error
				if err == nil {
					// Read into buffer
					_, readErr = io.ReadFull(resp.Body, data)
					_ = resp.Body.Close()
				}

				// Send result
				res := &chunk{
					id:   chunkID,
					data: data,
				}
				if err != nil {
					res.err = err
				} else if readErr != nil {
					res.err = readErr
				}

				select {
				case results <- res:
				case <-cr.ctx.Done():
					return
				}
			}
		}()
	}

	// Job feeder goroutine
	// Feeds jobs but limits how far ahead we go to avoid memory explosion
	go func() {
		defer close(jobs)

		// Max readahead chunks
		// If each chunk is 8MB, and concurrency is 16.
		// Max memory usage = (concurrency + lookahead) * 8MB
		// Keep lookahead small.
		lookAhead := cr.concurrency * 2
		_ = lookAhead // Explicitly ignore unused variable if just for documentation
		// We need to know how many chunks have been consumed by the writer to know if we can dispatch more.
		// However, tracking that here is hard.
		// Simplification: Just feed all jobs to channel BUT the channel is unbuffered (0).
		// Wait, make(chan int) is unbuffered.
		// So we can only feed if a worker is ready to take it.
		// A worker is ready only if it's not blocked on `results <- res`.
		// `results` has limited buffer.
		// So inherently this acts as backpressure!
		// If `results` is full (main thread blocked processing), workers block on send.
		// Then they don't pick new jobs.
		// Then `jobs <- i` blocks.
		// Perfect.

		for i := 0; i < numChunks; i++ {
			select {
			case jobs <- i:
			case <-cr.ctx.Done():
				return
			}
		}
	}()

	// Result processor
	buffer := make(map[int]*chunk)
	nextChunkID := 0

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {
		if res.err != nil {
			_ = cr.w.CloseWithError(res.err)
			cr.cancel() // Stop everything
			return
		}

		buffer[res.id] = res

		// Deliver consecutive chunks
		for {
			next, ok := buffer[nextChunkID]
			if !ok {
				break
			}
			delete(buffer, nextChunkID)

			// Write to pipe
			if _, err := cr.w.Write(next.data); err != nil {
				// Reader closed or error
				cr.cancel()
				return
			}
			nextChunkID++
		}
	}
}
