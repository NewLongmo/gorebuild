package repository

import "testing"

func TestChildRechargeCost(t *testing.T) {
	tests := []struct {
		name       string
		amount     float64
		parentRate float64
		childRate  float64
		want       float64
	}{
		{name: "same rate", amount: 100, parentRate: 1, childRate: 1, want: 100},
		{name: "child higher rate costs less", amount: 100, parentRate: 1, childRate: 2, want: 50},
		{name: "child lower rate costs more", amount: 100, parentRate: 2, childRate: 1, want: 200},
		{name: "rounds to cents", amount: 10, parentRate: 1, childRate: 3, want: 3.33},
		{name: "zero rates default to one", amount: 10, parentRate: 0, childRate: 0, want: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := childRechargeCost(tt.amount, tt.parentRate, tt.childRate); got != tt.want {
				t.Fatalf("childRechargeCost() = %v, want %v", got, tt.want)
			}
		})
	}
}
