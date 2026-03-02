package ws

import (
	"testing"
)

func TestDeviceStatusParams_ToStatusFilters(t *testing.T) {
	// Test Case 1: All filters set to "true"
	t.Run("All Filters True", func(t *testing.T) {
		params := DeviceStatusParams{
			DeviceListParams: DeviceListParams{
				AuthorizedOnly: "true",
				FilterOnline:   "true",
				FilterADB:      "true",
				FilterUSB:      "true",
				FilterHasIP:    "true",
			},
		}

		filters := params.ToStatusFilters()

		if filters.AuthorizedOnly == nil || !*filters.AuthorizedOnly {
			t.Error("Expected AuthorizedOnly to be true")
		}
		if filters.FilterOnline == nil || !*filters.FilterOnline {
			t.Error("Expected FilterOnline to be true")
		}
		if filters.FilterADB == nil || !*filters.FilterADB {
			t.Error("Expected FilterADB to be true")
		}
		if filters.FilterUSB == nil || !*filters.FilterUSB {
			t.Error("Expected FilterUSB to be true")
		}
		if filters.FilterHasIP == nil || !*filters.FilterHasIP {
			t.Error("Expected FilterHasIP to be true")
		}
	})

	// Test Case 2: All filters set to "false"
	t.Run("All Filters False", func(t *testing.T) {
		params := DeviceStatusParams{
			DeviceListParams: DeviceListParams{
				AuthorizedOnly: "false",
				FilterOnline:   "false",
				FilterADB:      "false",
				FilterUSB:      "false",
				FilterHasIP:    "false",
			},
		}

		filters := params.ToStatusFilters()

		if filters.AuthorizedOnly == nil || *filters.AuthorizedOnly {
			t.Error("Expected AuthorizedOnly to be false")
		}
		if filters.FilterOnline == nil || *filters.FilterOnline {
			t.Error("Expected FilterOnline to be false")
		}
		if filters.FilterADB == nil || *filters.FilterADB {
			t.Error("Expected FilterADB to be false")
		}
		if filters.FilterUSB == nil || *filters.FilterUSB {
			t.Error("Expected FilterUSB to be false")
		}
		if filters.FilterHasIP == nil || *filters.FilterHasIP {
			t.Error("Expected FilterHasIP to be false")
		}
	})

	// Test Case 3: All filters Empty (nil)
	t.Run("All Filters Empty", func(t *testing.T) {
		params := DeviceStatusParams{}

		filters := params.ToStatusFilters()

		if filters.AuthorizedOnly != nil {
			t.Error("Expected AuthorizedOnly to be nil")
		}
		if filters.FilterOnline != nil {
			t.Error("Expected FilterOnline to be nil")
		}
		if filters.FilterADB != nil {
			t.Error("Expected FilterADB to be nil")
		}
		if filters.FilterUSB != nil {
			t.Error("Expected FilterUSB to be nil")
		}
		if filters.FilterHasIP != nil {
			t.Error("Expected FilterHasIP to be nil")
		}
	})

	// Test Case 4: Numeric Filters
	t.Run("Numeric Filters", func(t *testing.T) {
		val := 10
		params := DeviceStatusParams{
			BizOnlineGT: &val,
		}

		filters := params.ToStatusFilters()

		if filters.BizOnlineGT != 10 {
			t.Errorf("Expected BizOnlineGT to be 10, got %d", filters.BizOnlineGT)
		}
	})
}
