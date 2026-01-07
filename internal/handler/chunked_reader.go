package handler

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// AwsChunkedReader decodes aws-chunked encoded body.
// Format per chunk: hex-size;signature-params \r\n data \r\n
type AwsChunkedReader struct {
	reader *bufio.Reader
	left   int64 // bytes left in current chunk
	err    error
}

func NewAwsChunkedReader(r io.Reader) *AwsChunkedReader {
	return &AwsChunkedReader{
		reader: bufio.NewReader(r),
	}
}

func (c *AwsChunkedReader) Read(p []byte) (int, error) {
	if c.err != nil {
		return 0, c.err
	}

	if c.left == 0 {
		// Read header line
		line, err := c.reader.ReadString('\n')
		if err != nil {
			c.err = err
			return 0, err
		}

		// Parse hex size
		// format: <hex>;...
		parts := strings.SplitN(line, ";", 2)
		// If no semicolon, it might just be <hex>\r\n (standard chunked), though AWS usually has sig.
		// Trimming space handles \r\n
		sizeHex := strings.TrimSpace(parts[0])

		size, err := strconv.ParseInt(sizeHex, 16, 64)
		if err != nil {
			c.err = fmt.Errorf("malformed chunk size: %v", err)
			return 0, c.err
		}

		if size == 0 {
			c.err = io.EOF
			return 0, io.EOF
		}

		c.left = size
	}

	readLen := int64(len(p))
	if readLen > c.left {
		readLen = c.left
	}

	n, err := c.reader.Read(p[:readLen])
	c.left -= int64(n)

	if err != nil && err != io.EOF {
		c.err = err
		return n, err
	}

	if c.left == 0 {
		// Consume \r\n
		_, discardErr := c.reader.Discard(2)
		if discardErr != nil {
			c.err = discardErr
			return n, discardErr
		}
	} else if err == io.EOF {
		c.err = io.ErrUnexpectedEOF
		return n, io.ErrUnexpectedEOF
	}

	return n, nil
}
