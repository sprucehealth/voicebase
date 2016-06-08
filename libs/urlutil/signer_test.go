package urlutil

import (
	"net/url"
	"path"
	"testing"
	"time"

	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/test"
)

func TestSignerRoundTrip(t *testing.T) {
	baseURL := "baseURL"
	sigKeys := [][]byte{[]byte("test_key")}
	startTime := time.Now()
	clk := clock.NewManaged(startTime)
	managedClocks := make(map[string]*clock.ManagedClock)
	cases := map[string]struct {
		Signer         *Signer
		Path           string
		Params         url.Values
		Expires        *time.Time
		PreFn          func()
		PostFn         func()
		ValidatePath   string
		ValidateParams url.Values
		Expected       error
	}{
		"ValidNonExpiring": {
			Signer: NewSigner(baseURL, func() *sig.Signer {
				s, err := sig.NewSigner(sigKeys, nil)
				test.OK(t, err)
				return s
			}(), clk),
			Path: "/path",
			Params: url.Values{
				"a": []string{"b", "c"},
			},
			Expires:      nil,
			ValidatePath: "/path",
			ValidateParams: url.Values{
				"a": []string{"b", "c"},
			},
			Expected: nil,
		},
		"ValidExpiring": {
			Signer: NewSigner(baseURL, func() *sig.Signer {
				s, err := sig.NewSigner(sigKeys, nil)
				test.OK(t, err)
				return s
			}(), clk),
			Path: "/path",
			Params: url.Values{
				"a": []string{"b", "c"},
			},
			Expires:      ptr.Time(startTime.Add(time.Minute * 15)),
			ValidatePath: "/path",
			ValidateParams: url.Values{
				"a": []string{"b", "c"},
			},
			Expected: nil,
		},
		"ValidDeterministicSorting": {
			Signer: NewSigner(baseURL, func() *sig.Signer {
				s, err := sig.NewSigner(sigKeys, nil)
				test.OK(t, err)
				return s
			}(), clk),
			Path: "/path",
			Params: url.Values{
				"d": []string{"f", "e"},
				"a": []string{"c", "b"},
			},
			Expires:      nil,
			ValidatePath: "/path",
			ValidateParams: url.Values{
				"a": []string{"c", "b"},
				"d": []string{"f", "e"},
			},
			Expected: nil,
		},
		"InvalidPathMismatch": {
			Signer: NewSigner(baseURL, func() *sig.Signer {
				s, err := sig.NewSigner(sigKeys, nil)
				test.OK(t, err)
				return s
			}(), clk),
			Path: "/path",
			Params: url.Values{
				"a": []string{"b", "c"},
			},
			Expires:      nil,
			ValidatePath: "/path/more",
			ValidateParams: url.Values{
				"a": []string{"b", "c"},
			},
			Expected: ErrSignatureMismatch,
		},
		"InvalidParamValuesMismatch": {
			Signer: NewSigner(baseURL, func() *sig.Signer {
				s, err := sig.NewSigner(sigKeys, nil)
				test.OK(t, err)
				return s
			}(), clk),
			Path: "/path",
			Params: url.Values{
				"a": []string{"b"},
			},
			Expires:      nil,
			ValidatePath: "/path",
			ValidateParams: url.Values{
				"a": []string{"b", "c"},
			},
			Expected: ErrSignatureMismatch,
		},
		"InvalidParamNameMismatch": {
			Signer: NewSigner(baseURL, func() *sig.Signer {
				s, err := sig.NewSigner(sigKeys, nil)
				test.OK(t, err)
				return s
			}(), clk),
			Path: "/path",
			Params: url.Values{
				"d": []string{"b", "c"},
			},
			Expires:      nil,
			ValidatePath: "/path",
			ValidateParams: url.Values{
				"a": []string{"b", "c"},
			},
			Expected: ErrSignatureMismatch,
		},
		"InvalidExpired": {
			Signer: NewSigner(baseURL, func() *sig.Signer {
				s, err := sig.NewSigner(sigKeys, nil)
				test.OK(t, err)
				return s
			}(), func() clock.Clock {
				managedClocks["InvalidExpired"] = clock.NewManaged(startTime)
				return managedClocks["InvalidExpired"]
			}()),
			Path: "/path",
			Params: url.Values{
				"a": []string{"b", "c"},
			},
			PreFn: func() {
				managedClocks["InvalidExpired"].WarpForward(time.Minute * 16)
			},
			Expires:      ptr.Time(startTime.Add(time.Minute * 15)),
			ValidatePath: "/path",
			ValidateParams: url.Values{
				"a": []string{"b", "c"},
			},
			Expected: ErrExpiredURL,
		},
	}

	for cn, c := range cases {
		if c.PreFn != nil {
			c.PreFn()
		}
		u, err := c.Signer.SignedURL(c.Path, c.Params, c.Expires)
		test.OKCase(t, cn, err)
		if c.PostFn != nil {
			c.PostFn()
		}
		ru, err := url.Parse(u)
		test.OKCase(t, cn, err)
		// Assert that the generated path matches the input
		test.EqualsCase(t, cn, path.Join(baseURL, c.Path), ru.Path)
		ps, err := url.ParseQuery(ru.RawQuery)
		test.OKCase(t, cn, err)
		// Copy the sig into the validation set
		c.ValidateParams.Set(SigParamName, ps.Get(SigParamName))
		if ps.Get(expiresParamName) != "" {
			c.ValidateParams.Set(expiresParamName, ps.Get(expiresParamName))
		}
		test.EqualsCase(t, cn, c.Expected, c.Signer.ValidateSignature(c.ValidatePath, c.ValidateParams))
	}
}
