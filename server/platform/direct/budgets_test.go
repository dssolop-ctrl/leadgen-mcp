package direct

import "testing"

// TestMinWeeklyBudgetFloor verifies the floor lookup used by add_campaign's
// budget gate. Floors are: search=5000, rsya=3000, vk=2000.
func TestMinWeeklyBudgetFloor(t *testing.T) {
	cases := []struct {
		channel string
		want    int
	}{
		{"search", 5000},
		{"SEARCH", 5000},  // case-insensitive
		{" search ", 5000}, // trim
		{"rsya", 3000},
		{"RSYA", 3000},
		{"vk", 2000},
		{"unknown", 0},
		{"", 0},
	}
	for _, c := range cases {
		got := MinWeeklyBudgetFloor(c.channel)
		if got != c.want {
			t.Errorf("MinWeeklyBudgetFloor(%q) = %d, want %d", c.channel, got, c.want)
		}
	}
}
