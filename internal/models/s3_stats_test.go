package models

import (
	"testing"
)

// TestS3MethodStats tests individual method statistics tracking
func TestS3MethodStats(t *testing.T) {
	ms := &S3MethodStats{}

	// Record some successes
	ms.RecordSuccess()
	ms.RecordSuccess()
	ms.RecordSuccess()

	// Record some failures
	ms.RecordFailure()
	ms.RecordFailure()

	// Get stats
	successes, failures := ms.GetStats()

	if successes != 3 {
		t.Errorf("expected 3 successes, got %d", successes)
	}
	if failures != 2 {
		t.Errorf("expected 2 failures, got %d", failures)
	}

	// Reset
	ms.Reset()
	successes, failures = ms.GetStats()

	if successes != 0 || failures != 0 {
		t.Errorf("expected all zeros after reset, got successes=%d failures=%d", successes, failures)
	}
}

// TestS3OperationStats tests overall S3 operation statistics
func TestS3OperationStats(t *testing.T) {
	stats := &S3OperationStats{}

	// Record some operations
	stats.RecordListBuckets(true)
	stats.RecordListBuckets(false)

	stats.RecordCreateBucket(true)
	stats.RecordCreateBucket(true)

	stats.RecordDeleteBucket(false)

	stats.RecordGetObject(true)
	stats.RecordGetObject(true)
	stats.RecordGetObject(true)

	stats.RecordPutObject(true)
	stats.RecordPutObject(false)

	// Get stats snapshot
	snap := stats.GetStats()

	// Verify aggregated totals
	// Successes: 1 ListBuckets + 2 CreateBucket + 3 GetObject + 1 PutObject = 7
	// Failures: 1 ListBuckets + 1 DeleteBucket + 1 PutObject = 3
	// Total = 10
	if snap.TotalSuccesses != 7 {
		t.Errorf("expected 7 total successes, got %d", snap.TotalSuccesses)
	}
	if snap.TotalFailures != 3 {
		t.Errorf("expected 3 total failures, got %d", snap.TotalFailures)
	}
	if snap.TotalRequests != 10 {
		t.Errorf("expected 10 total requests, got %d", snap.TotalRequests)
	}

	// Verify success rate (7/10 = 70%)
	expectedRate := (7.0 / 10.0) * 100
	if snap.OverallSuccessRate != expectedRate {
		t.Errorf("expected %.2f%% success rate, got %.2f%%", expectedRate, snap.OverallSuccessRate)
	}

	// Verify per-method stats
	if snap.Methods["ListBuckets"].Success != 1 {
		t.Errorf("expected 1 ListBuckets success, got %d", snap.Methods["ListBuckets"].Success)
	}
	if snap.Methods["ListBuckets"].Failure != 1 {
		t.Errorf("expected 1 ListBuckets failure, got %d", snap.Methods["ListBuckets"].Failure)
	}

	if snap.Methods["CreateBucket"].Success != 2 {
		t.Errorf("expected 2 CreateBucket successes, got %d", snap.Methods["CreateBucket"].Success)
	}
	if snap.Methods["CreateBucket"].Failure != 0 {
		t.Errorf("expected 0 CreateBucket failures, got %d", snap.Methods["CreateBucket"].Failure)
	}

	if snap.Methods["GetObject"].Success != 3 {
		t.Errorf("expected 3 GetObject successes, got %d", snap.Methods["GetObject"].Success)
	}

	if snap.Methods["PutObject"].Success != 1 {
		t.Errorf("expected 1 PutObject success, got %d", snap.Methods["PutObject"].Success)
	}
	if snap.Methods["PutObject"].Failure != 1 {
		t.Errorf("expected 1 PutObject failure, got %d", snap.Methods["PutObject"].Failure)
	}
}

// TestS3OperationStats_SuccessRate tests per-method success rate calculation
func TestS3OperationStats_SuccessRate(t *testing.T) {
	stats := &S3OperationStats{}

	// Test with 100% success rate
	stats.RecordHeadObject(true)
	stats.RecordHeadObject(true)
	stats.RecordHeadObject(true)

	snap := stats.GetStats()
	if snap.Methods["HeadObject"].SuccessRate != 100.0 {
		t.Errorf("expected 100%% success rate for HeadObject, got %.2f%%", snap.Methods["HeadObject"].SuccessRate)
	}

	// Test with 0% success rate
	stats2 := &S3OperationStats{}
	stats2.RecordDeleteObject(false)
	stats2.RecordDeleteObject(false)

	snap2 := stats2.GetStats()
	if snap2.Methods["DeleteObject"].SuccessRate != 0.0 {
		t.Errorf("expected 0%% success rate for DeleteObject, got %.2f%%", snap2.Methods["DeleteObject"].SuccessRate)
	}

	// Test with 50% success rate
	stats3 := &S3OperationStats{}
	stats3.RecordUploadPart(true)
	stats3.RecordUploadPart(false)

	snap3 := stats3.GetStats()
	if snap3.Methods["UploadPart"].SuccessRate != 50.0 {
		t.Errorf("expected 50%% success rate for UploadPart, got %.2f%%", snap3.Methods["UploadPart"].SuccessRate)
	}
}

// TestS3OperationStats_AllMethods tests that all methods are tracked
func TestS3OperationStats_AllMethods(t *testing.T) {
	stats := &S3OperationStats{}

	// Record one operation for each method
	stats.RecordListBuckets(true)
	stats.RecordCreateBucket(true)
	stats.RecordDeleteBucket(true)
	stats.RecordListObjectsV2(true)
	stats.RecordGetObject(true)
	stats.RecordPutObject(true)
	stats.RecordDeleteObject(true)
	stats.RecordHeadObject(true)
	stats.RecordDeleteMultiple(true)
	stats.RecordInitiateMultipart(true)
	stats.RecordUploadPart(true)
	stats.RecordCompleteMultipart(true)
	stats.RecordAbortMultipart(true)
	stats.RecordListParts(true)
	stats.RecordListMultipartUploads(true)
	stats.RecordEnableVersioning(true)
	stats.RecordGetVersioning(true)
	stats.RecordListObjectVersions(true)
	stats.RecordGetObjectVersion(true)
	stats.RecordDeleteObjectVersion(true)

	snap := stats.GetStats()

	// Verify all 20 methods are present
	expectedMethods := []string{
		"ListBuckets",
		"CreateBucket",
		"DeleteBucket",
		"ListObjectsV2",
		"GetObject",
		"PutObject",
		"DeleteObject",
		"HeadObject",
		"DeleteMultipleObjects",
		"InitiateMultipartUpload",
		"UploadPart",
		"CompleteMultipartUpload",
		"AbortMultipartUpload",
		"ListParts",
		"ListMultipartUploads",
		"EnableVersioning",
		"GetVersioning",
		"ListObjectVersions",
		"GetObjectVersion",
		"DeleteObjectVersion",
	}

	for _, methodName := range expectedMethods {
		if _, exists := snap.Methods[methodName]; !exists {
			t.Errorf("method %s not found in stats snapshot", methodName)
		}
		if snap.Methods[methodName].Success != 1 {
			t.Errorf("expected 1 success for %s, got %d", methodName, snap.Methods[methodName].Success)
		}
	}

	// Verify total counts
	if snap.TotalSuccesses != 20 {
		t.Errorf("expected 20 total successes, got %d", snap.TotalSuccesses)
	}
	if snap.TotalFailures != 0 {
		t.Errorf("expected 0 total failures, got %d", snap.TotalFailures)
	}
	if snap.OverallSuccessRate != 100.0 {
		t.Errorf("expected 100%% overall success rate, got %.2f%%", snap.OverallSuccessRate)
	}
}

// TestS3OperationStats_Reset tests reset functionality
func TestS3OperationStats_Reset(t *testing.T) {
	stats := &S3OperationStats{}

	// Record operations
	stats.RecordListBuckets(true)
	stats.RecordCreateBucket(false)
	stats.RecordGetObject(true)

	snap := stats.GetStats()
	if snap.TotalRequests != 3 {
		t.Errorf("expected 3 requests before reset, got %d", snap.TotalRequests)
	}

	// Reset
	stats.Reset()

	snap = stats.GetStats()
	if snap.TotalSuccesses != 0 {
		t.Errorf("expected 0 successes after reset, got %d", snap.TotalSuccesses)
	}
	if snap.TotalFailures != 0 {
		t.Errorf("expected 0 failures after reset, got %d", snap.TotalFailures)
	}
	if snap.TotalRequests != 0 {
		t.Errorf("expected 0 requests after reset, got %d", snap.TotalRequests)
	}

	// All methods should have zero counts
	for _, method := range snap.Methods {
		if method.Success != 0 || method.Failure != 0 {
			t.Errorf("method %s not reset: success=%d, failure=%d", method.Name, method.Success, method.Failure)
		}
	}
}

// TestS3OperationStats_Concurrent tests thread-safe concurrent updates
func TestS3OperationStats_Concurrent(t *testing.T) {
	stats := &S3OperationStats{}

	// Run concurrent operations
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				switch j % 3 {
				case 0:
					stats.RecordListBuckets(true)
				case 1:
					stats.RecordListBuckets(false)
				default:
					stats.RecordGetObject(true)
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	snap := stats.GetStats()

	// With 10 goroutines and 100 iterations each:
	// ~333 ListBuckets successes
	// ~334 ListBuckets failures
	// ~333 GetObject successes
	// Total: ~1000 operations

	if snap.TotalRequests != 1000 {
		t.Errorf("expected 1000 total requests, got %d", snap.TotalRequests)
	}

	// Verify ListBuckets has roughly 667 operations (2/3 of 1000)
	listBucketsTotal := snap.Methods["ListBuckets"].Total
	if listBucketsTotal < 650 || listBucketsTotal > 680 {
		t.Errorf("expected ~667 ListBuckets operations, got %d", listBucketsTotal)
	}

	// Verify GetObject has roughly 333 operations (1/3 of 1000)
	getObjectTotal := snap.Methods["GetObject"].Total
	if getObjectTotal < 310 || getObjectTotal > 340 {
		t.Errorf("expected ~333 GetObject operations, got %d", getObjectTotal)
	}
}

// TestAllRecordMethods tests all Record* methods to ensure complete coverage
func TestAllRecordMethods(t *testing.T) {
	stats := &S3OperationStats{}

	// Test all Record methods with both success and failure
	stats.RecordListObjectsV2(true)
	stats.RecordListObjectsV2(false)

	stats.RecordGetObject(true)
	stats.RecordGetObject(false)

	stats.RecordHeadObject(true)
	stats.RecordHeadObject(false)

	stats.RecordDeleteMultiple(true)
	stats.RecordDeleteMultiple(false)

	stats.RecordInitiateMultipart(true)
	stats.RecordInitiateMultipart(false)

	stats.RecordCompleteMultipart(true)
	stats.RecordCompleteMultipart(false)

	stats.RecordAbortMultipart(true)
	stats.RecordAbortMultipart(false)

	stats.RecordListParts(true)
	stats.RecordListParts(false)

	stats.RecordListMultipartUploads(true)
	stats.RecordListMultipartUploads(false)

	stats.RecordEnableVersioning(true)
	stats.RecordEnableVersioning(false)

	stats.RecordGetVersioning(true)
	stats.RecordGetVersioning(false)

	stats.RecordListObjectVersions(true)
	stats.RecordListObjectVersions(false)

	stats.RecordGetObjectVersion(true)
	stats.RecordGetObjectVersion(false)

	stats.RecordDeleteObjectVersion(true)
	stats.RecordDeleteObjectVersion(false)

	// Get the snapshot and verify all methods were recorded
	snap := stats.GetStats()

	expectedMethods := []string{
		"ListObjectsV2", "GetObject", "HeadObject", "DeleteMultipleObjects",
		"InitiateMultipartUpload", "CompleteMultipartUpload", "AbortMultipartUpload",
		"ListParts", "ListMultipartUploads", "EnableVersioning",
		"GetVersioning", "ListObjectVersions", "GetObjectVersion",
		"DeleteObjectVersion",
	}

	// Each method should have 2 total (1 success + 1 failure)
	for _, methodName := range expectedMethods {
		if methodStats, exists := snap.Methods[methodName]; exists {
			if methodStats.Total != 2 {
				t.Errorf("%s: expected total=2, got %d", methodName, methodStats.Total)
			}
			if methodStats.Success != 1 {
				t.Errorf("%s: expected success=1, got %d", methodName, methodStats.Success)
			}
			if methodStats.Failure != 1 {
				t.Errorf("%s: expected failure=1, got %d", methodName, methodStats.Failure)
			}
			if methodStats.SuccessRate != 50.0 {
				t.Errorf("%s: expected success_rate=50.0, got %f", methodName, methodStats.SuccessRate)
			}
		} else {
			t.Errorf("Method %s not found in snapshot", methodName)
		}
	}

	// Verify overall stats
	expectedTotal := len(expectedMethods) * 2
	if snap.TotalRequests != int64(expectedTotal) {
		t.Errorf("expected TotalRequests=%d, got %d", expectedTotal, snap.TotalRequests)
	}

	if snap.TotalSuccesses != int64(len(expectedMethods)) {
		t.Errorf("expected TotalSuccesses=%d, got %d", len(expectedMethods), snap.TotalSuccesses)
	}

	if snap.TotalFailures != int64(len(expectedMethods)) {
		t.Errorf("expected TotalFailures=%d, got %d", len(expectedMethods), snap.TotalFailures)
	}
}
