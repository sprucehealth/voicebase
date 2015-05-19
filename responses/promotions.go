package responses

type Promotion struct {
	Code                 string `json:"display_code"`
	Description          string `json:"description"`
	DescriptionHasTokens bool   `json:"description_has_tokens"`
	ExpirationDate       int64  `json:"expiration_date,string"`
}
