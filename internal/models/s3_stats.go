package models

import (
	"sync/atomic"
)

// S3MethodStats tracks statistics for individual S3 API methods.
// Each method has success and failure counters.
type S3MethodStats struct {
	successes atomic.Int64 // Count of successful method calls
	failures  atomic.Int64 // Count of failed method calls
}

// RecordSuccess increments the success counter for the method.
func (ms *S3MethodStats) RecordSuccess() {
	ms.successes.Add(1)
}

// RecordFailure increments the failure counter for the method.
func (ms *S3MethodStats) RecordFailure() {
	ms.failures.Add(1)
}

// GetStats returns a snapshot of the method statistics.
func (ms *S3MethodStats) GetStats() (successes, failures int64) {
	return ms.successes.Load(), ms.failures.Load()
}

// Reset clears all counters.
func (ms *S3MethodStats) Reset() {
	ms.successes.Store(0)
	ms.failures.Store(0)
}

// S3OperationStats tracks comprehensive S3 operation statistics.
// It maintains per-method counters and aggregated success/failure metrics.
type S3OperationStats struct {
	// Per-method statistics
	listBuckets      S3MethodStats // ListBuckets operation
	createBucket     S3MethodStats // CreateBucket operation
	deleteBucket     S3MethodStats // DeleteBucket operation
	listObjectsV2    S3MethodStats // ListObjectsV2 operation
	getObject        S3MethodStats // GetObject operation
	putObject        S3MethodStats // PutObject operation
	deleteObject     S3MethodStats // DeleteObject operation
	headObject       S3MethodStats // HeadObject operation
	deleteMultiple   S3MethodStats // DeleteMultipleObjects operation
	initiateMulti    S3MethodStats // InitiateMultipartUpload operation
	uploadPart       S3MethodStats // UploadPart operation
	completeMulti    S3MethodStats // CompleteMultipartUpload operation
	abortMulti       S3MethodStats // AbortMultipartUpload operation
	listParts        S3MethodStats // ListParts operation
	listMultiUploads S3MethodStats // ListMultipartUploads operation
	enableVersion    S3MethodStats // EnableVersioning operation
	getVersion       S3MethodStats // GetVersioning operation
	listObjVersions  S3MethodStats // ListObjectVersions operation
	getObjVersion    S3MethodStats // GetObjectVersion operation
	deleteObjVersion S3MethodStats // DeleteObjectVersion operation

	// Aggregated totals
	totalSuccesses atomic.Int64 // Total successful operations
	totalFailures  atomic.Int64 // Total failed operations
}

// RecordListBuckets records ListBuckets operation result.
func (s *S3OperationStats) RecordListBuckets(success bool) {
	if success {
		s.listBuckets.RecordSuccess()
		s.totalSuccesses.Add(1)
	} else {
		s.listBuckets.RecordFailure()
		s.totalFailures.Add(1)
	}
}

// RecordCreateBucket records CreateBucket operation result.
func (s *S3OperationStats) RecordCreateBucket(success bool) {
	if success {
		s.createBucket.RecordSuccess()
		s.totalSuccesses.Add(1)
	} else {
		s.createBucket.RecordFailure()
		s.totalFailures.Add(1)
	}
}

// RecordDeleteBucket records DeleteBucket operation result.
func (s *S3OperationStats) RecordDeleteBucket(success bool) {
	if success {
		s.deleteBucket.RecordSuccess()
		s.totalSuccesses.Add(1)
	} else {
		s.deleteBucket.RecordFailure()
		s.totalFailures.Add(1)
	}
}

// RecordListObjectsV2 records ListObjectsV2 operation result.
func (s *S3OperationStats) RecordListObjectsV2(success bool) {
	if success {
		s.listObjectsV2.RecordSuccess()
		s.totalSuccesses.Add(1)
	} else {
		s.listObjectsV2.RecordFailure()
		s.totalFailures.Add(1)
	}
}

// RecordGetObject records GetObject operation result.
func (s *S3OperationStats) RecordGetObject(success bool) {
	if success {
		s.getObject.RecordSuccess()
		s.totalSuccesses.Add(1)
	} else {
		s.getObject.RecordFailure()
		s.totalFailures.Add(1)
	}
}

// RecordPutObject records PutObject operation result.
func (s *S3OperationStats) RecordPutObject(success bool) {
	if success {
		s.putObject.RecordSuccess()
		s.totalSuccesses.Add(1)
	} else {
		s.putObject.RecordFailure()
		s.totalFailures.Add(1)
	}
}

// RecordDeleteObject records DeleteObject operation result.
func (s *S3OperationStats) RecordDeleteObject(success bool) {
	if success {
		s.deleteObject.RecordSuccess()
		s.totalSuccesses.Add(1)
	} else {
		s.deleteObject.RecordFailure()
		s.totalFailures.Add(1)
	}
}

// RecordHeadObject records HeadObject operation result.
func (s *S3OperationStats) RecordHeadObject(success bool) {
	if success {
		s.headObject.RecordSuccess()
		s.totalSuccesses.Add(1)
	} else {
		s.headObject.RecordFailure()
		s.totalFailures.Add(1)
	}
}

// RecordDeleteMultiple records DeleteMultipleObjects operation result.
func (s *S3OperationStats) RecordDeleteMultiple(success bool) {
	if success {
		s.deleteMultiple.RecordSuccess()
		s.totalSuccesses.Add(1)
	} else {
		s.deleteMultiple.RecordFailure()
		s.totalFailures.Add(1)
	}
}

// RecordInitiateMultipart records InitiateMultipartUpload operation result.
func (s *S3OperationStats) RecordInitiateMultipart(success bool) {
	if success {
		s.initiateMulti.RecordSuccess()
		s.totalSuccesses.Add(1)
	} else {
		s.initiateMulti.RecordFailure()
		s.totalFailures.Add(1)
	}
}

// RecordUploadPart records UploadPart operation result.
func (s *S3OperationStats) RecordUploadPart(success bool) {
	if success {
		s.uploadPart.RecordSuccess()
		s.totalSuccesses.Add(1)
	} else {
		s.uploadPart.RecordFailure()
		s.totalFailures.Add(1)
	}
}

// RecordCompleteMultipart records CompleteMultipartUpload operation result.
func (s *S3OperationStats) RecordCompleteMultipart(success bool) {
	if success {
		s.completeMulti.RecordSuccess()
		s.totalSuccesses.Add(1)
	} else {
		s.completeMulti.RecordFailure()
		s.totalFailures.Add(1)
	}
}

// RecordAbortMultipart records AbortMultipartUpload operation result.
func (s *S3OperationStats) RecordAbortMultipart(success bool) {
	if success {
		s.abortMulti.RecordSuccess()
		s.totalSuccesses.Add(1)
	} else {
		s.abortMulti.RecordFailure()
		s.totalFailures.Add(1)
	}
}

// RecordListParts records ListParts operation result.
func (s *S3OperationStats) RecordListParts(success bool) {
	if success {
		s.listParts.RecordSuccess()
		s.totalSuccesses.Add(1)
	} else {
		s.listParts.RecordFailure()
		s.totalFailures.Add(1)
	}
}

// RecordListMultipartUploads records ListMultipartUploads operation result.
func (s *S3OperationStats) RecordListMultipartUploads(success bool) {
	if success {
		s.listMultiUploads.RecordSuccess()
		s.totalSuccesses.Add(1)
	} else {
		s.listMultiUploads.RecordFailure()
		s.totalFailures.Add(1)
	}
}

// RecordEnableVersioning records EnableVersioning operation result.
func (s *S3OperationStats) RecordEnableVersioning(success bool) {
	if success {
		s.enableVersion.RecordSuccess()
		s.totalSuccesses.Add(1)
	} else {
		s.enableVersion.RecordFailure()
		s.totalFailures.Add(1)
	}
}

// RecordGetVersioning records GetVersioning operation result.
func (s *S3OperationStats) RecordGetVersioning(success bool) {
	if success {
		s.getVersion.RecordSuccess()
		s.totalSuccesses.Add(1)
	} else {
		s.getVersion.RecordFailure()
		s.totalFailures.Add(1)
	}
}

// RecordListObjectVersions records ListObjectVersions operation result.
func (s *S3OperationStats) RecordListObjectVersions(success bool) {
	if success {
		s.listObjVersions.RecordSuccess()
		s.totalSuccesses.Add(1)
	} else {
		s.listObjVersions.RecordFailure()
		s.totalFailures.Add(1)
	}
}

// RecordGetObjectVersion records GetObjectVersion operation result.
func (s *S3OperationStats) RecordGetObjectVersion(success bool) {
	if success {
		s.getObjVersion.RecordSuccess()
		s.totalSuccesses.Add(1)
	} else {
		s.getObjVersion.RecordFailure()
		s.totalFailures.Add(1)
	}
}

// RecordDeleteObjectVersion records DeleteObjectVersion operation result.
func (s *S3OperationStats) RecordDeleteObjectVersion(success bool) {
	if success {
		s.deleteObjVersion.RecordSuccess()
		s.totalSuccesses.Add(1)
	} else {
		s.deleteObjVersion.RecordFailure()
		s.totalFailures.Add(1)
	}
}

// MethodStatsSnapshot holds a snapshot of per-method statistics.
type MethodStatsSnapshot struct {
	Name        string
	Success     int64
	Failure     int64
	Total       int64
	SuccessRate float64 // Percentage 0-100
}

// StatsSnapshot holds a snapshot of all S3 operation statistics.
type StatsSnapshot struct {
	// Per-method statistics
	Methods map[string]MethodStatsSnapshot

	// Aggregated statistics
	TotalSuccesses     int64
	TotalFailures      int64
	TotalRequests      int64
	OverallSuccessRate float64 // Percentage 0-100
}

// GetStats returns a snapshot of all S3 operation statistics.
func (s *S3OperationStats) GetStats() StatsSnapshot {
	// Collect per-method stats
	methods := make(map[string]MethodStatsSnapshot)

	methodStats := []struct {
		name  string
		stats *S3MethodStats
	}{
		{"ListBuckets", &s.listBuckets},
		{"CreateBucket", &s.createBucket},
		{"DeleteBucket", &s.deleteBucket},
		{"ListObjectsV2", &s.listObjectsV2},
		{"GetObject", &s.getObject},
		{"PutObject", &s.putObject},
		{"DeleteObject", &s.deleteObject},
		{"HeadObject", &s.headObject},
		{"DeleteMultipleObjects", &s.deleteMultiple},
		{"InitiateMultipartUpload", &s.initiateMulti},
		{"UploadPart", &s.uploadPart},
		{"CompleteMultipartUpload", &s.completeMulti},
		{"AbortMultipartUpload", &s.abortMulti},
		{"ListParts", &s.listParts},
		{"ListMultipartUploads", &s.listMultiUploads},
		{"EnableVersioning", &s.enableVersion},
		{"GetVersioning", &s.getVersion},
		{"ListObjectVersions", &s.listObjVersions},
		{"GetObjectVersion", &s.getObjVersion},
		{"DeleteObjectVersion", &s.deleteObjVersion},
	}

	for _, ms := range methodStats {
		success, failure := ms.stats.GetStats()
		total := success + failure
		successRate := float64(0)
		if total > 0 {
			successRate = (float64(success) / float64(total)) * 100
		}

		methods[ms.name] = MethodStatsSnapshot{
			Name:        ms.name,
			Success:     success,
			Failure:     failure,
			Total:       total,
			SuccessRate: successRate,
		}
	}

	// Calculate overall statistics
	totalSuccesses := s.totalSuccesses.Load()
	totalFailures := s.totalFailures.Load()
	totalRequests := totalSuccesses + totalFailures
	overallSuccessRate := float64(0)
	if totalRequests > 0 {
		overallSuccessRate = (float64(totalSuccesses) / float64(totalRequests)) * 100
	}

	return StatsSnapshot{
		Methods:            methods,
		TotalSuccesses:     totalSuccesses,
		TotalFailures:      totalFailures,
		TotalRequests:      totalRequests,
		OverallSuccessRate: overallSuccessRate,
	}
}

// Reset clears all statistics counters.
func (s *S3OperationStats) Reset() {
	s.listBuckets.Reset()
	s.createBucket.Reset()
	s.deleteBucket.Reset()
	s.listObjectsV2.Reset()
	s.getObject.Reset()
	s.putObject.Reset()
	s.deleteObject.Reset()
	s.headObject.Reset()
	s.deleteMultiple.Reset()
	s.initiateMulti.Reset()
	s.uploadPart.Reset()
	s.completeMulti.Reset()
	s.abortMulti.Reset()
	s.listParts.Reset()
	s.listMultiUploads.Reset()
	s.enableVersion.Reset()
	s.getVersion.Reset()
	s.listObjVersions.Reset()
	s.getObjVersion.Reset()
	s.deleteObjVersion.Reset()
	s.totalSuccesses.Store(0)
	s.totalFailures.Store(0)
}
