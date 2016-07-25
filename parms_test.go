// Copyright 2016 Martin Angers. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpparms

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/go-playground/form"
	"github.com/gorilla/schema"
	"github.com/pquerna/ffjson/ffjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type parmTest struct {
	S string `schema:"s" form:"s"`
	I int    `schema:"i" form:"i"`
	Q string `schema:":q" form:":q" json:":q"`
}

func (pt *parmTest) Validate() error {
	if pt.I > 2 {
		return errors.New("too big")
	}
	return nil
}

func TestParseQueryForm(t *testing.T) {
	cases := []struct {
		parms url.Values
		want  parmTest
		err   bool
	}{
		{url.Values{}, parmTest{}, false},
		{url.Values{"s": {"a"}}, parmTest{S: "a"}, false},
		{url.Values{"i": {"9"}}, parmTest{I: 9}, true},
		{url.Values{"i": {"1"}}, parmTest{I: 1}, false},
		{url.Values{"i": {"X"}}, parmTest{}, true},
		{url.Values{"s": {"X"}, "i": {"-1"}, ":q": {"q"}}, parmTest{S: "X", I: -1, Q: "q"}, false},
	}
	dec1 := schema.NewDecoder()
	dec1.IgnoreUnknownKeys(true)
	dec2 := form.NewDecoder()
	for j, fn := range []FormDecoder{dec1.Decode, FormDecoderAdapter(dec2.Decode)} {
		for i, c := range cases {
			var pt parmTest
			r, err := http.NewRequest("GET", "/a", nil)
			require.NoError(t, err)
			r.Form = c.parms

			p := &Parser{Form: fn}
			got := p.ParseQueryForm(r, &pt)
			assert.Equal(t, c.err, got != nil, "%d (%d): error expected?", i, j)
		}
	}
}

func TestParseJSON(t *testing.T) {
	cases := []struct {
		body string
		want parmTest
		err  bool
	}{
		{``, parmTest{}, false},
		{`"abc"`, parmTest{}, true},
		{`{"s": "X"}`, parmTest{S: "X"}, false},
		{`{"i": "X"}`, parmTest{}, true},
		{`{"i": 9}`, parmTest{I: 9}, true},
		{`{"s": "X", "i": 1, ":q": "Q"}`, parmTest{I: 1, S: "X", Q: "Q"}, false},
	}
	for j, fn := range []JSONUnmarshaler{json.Unmarshal, ffjson.Unmarshal} {
		for i, c := range cases {
			var pt parmTest
			r, err := http.NewRequest("GET", "/a", strings.NewReader(c.body))
			require.NoError(t, err)

			p := &Parser{JSON: fn}
			got := p.ParseJSON(r, &pt)
			if !assert.Equal(t, c.err, got != nil, "%d (%d): error expected?", i, j) {
				t.Logf("%d (%d): unexpected error: %v", i, j, got)
			}
		}
	}
}
