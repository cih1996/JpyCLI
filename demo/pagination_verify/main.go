package main

import (
	"fmt"
	"testing"
)

// Simulate the pagination logic from handlers.go
func paginate(items []int, page, pageSize int) []int {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}

	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= len(items) {
		return []int{}
	}
	if end > len(items) {
		end = len(items)
	}

	return items[start:end]
}

func TestPaginationLogic(t *testing.T) {
	// Create a list of 100 items
	items := make([]int, 100)
	for i := 0; i < 100; i++ {
		items[i] = i
	}

	tests := []struct {
		name     string
		page     int
		pageSize int
		wantLen  int
		wantStart int
		wantEnd   int // exclusive
	}{
		{"Page 1, Size 10", 1, 10, 10, 0, 10},
		{"Page 2, Size 10", 2, 10, 10, 10, 20},
		{"Page 10, Size 10", 10, 10, 10, 90, 100},
		{"Page 11, Size 10 (Empty)", 11, 10, 0, 0, 0},
		{"Page 1, Size 105 (All)", 1, 105, 100, 0, 100},
		{"Invalid Page (0 -> 1)", 0, 10, 10, 0, 10},
		{"Invalid Size (0 -> 50)", 1, 0, 50, 0, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := paginate(items, tt.page, tt.pageSize)
			if len(got) != tt.wantLen {
				t.Errorf("got length %d, want %d", len(got), tt.wantLen)
			}
			if tt.wantLen > 0 {
				if got[0] != tt.wantStart {
					t.Errorf("got start %d, want %d", got[0], tt.wantStart)
				}
				if got[len(got)-1] != tt.wantEnd-1 {
					t.Errorf("got end %d, want %d", got[len(got)-1], tt.wantEnd-1)
				}
			}
		})
	}
}

func main() {
	// Manually run the test function for demo purposes
	fmt.Println("Running Pagination Logic Verification...")
	
	items := make([]int, 100)
	for i := 0; i < 100; i++ {
		items[i] = i
	}

	// Case 1: Page 1, Size 20
	p1 := paginate(items, 1, 20)
	fmt.Printf("Page 1 (Size 20): Count=%d, Range=%d-%d\n", len(p1), p1[0], p1[len(p1)-1])

	// Case 2: Page 2, Size 20
	p2 := paginate(items, 2, 20)
	fmt.Printf("Page 2 (Size 20): Count=%d, Range=%d-%d\n", len(p2), p2[0], p2[len(p2)-1])
	
	// Case 3: Page 6 (Out of bounds)
	p6 := paginate(items, 6, 20)
	fmt.Printf("Page 6 (Size 20): Count=%d\n", len(p6))

	fmt.Println("Verification Complete.")
}
