package payments

import "github.com/dustin/go-humanize"

// FormatAmount formats the provided amount in the following format:
// If amount == 4600 then return $46
// If amount = 460 then return $4.60
// If amount = 461 then return $4.61
// If amount = 461000 then return $4,610
func FormatAmount(amount uint64, currency string) string {

	if amount%100 == 0 {
		return "$" + humanize.Commaf(float64(amount)/float64(100.0))
	}

	return "$" + humanize.FormatFloat("", float64(amount)/float64(100.0))
}
