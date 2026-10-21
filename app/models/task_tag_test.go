package models

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseTaskTags(t *testing.T) {
	cases := []struct {
		name, input string
		want        []string
		invalid     bool
	}{
		{"empty", " ,，\n", []string{}, false},
		{"normalize", " 生产 , 数据同步，生产\r\n日报 ", []string{"生产", "数据同步", "日报"}, false},
		{"exact names", "prod,production,Prod,a&b", []string{"prod", "production", "Prod", "a&b"}, false},
		{"unicode limit", strings.Repeat("标", 30), []string{strings.Repeat("标", 30)}, false},
		{"too long", strings.Repeat("标", 31), nil, true},
		{"too many", "a,b,c,d,e,f,g,h,i,j,k,l,m,n,o,p,q,r,s,t,u", nil, true},
		{"duplicates do not count", strings.Repeat("a,", 30), []string{"a"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseTaskTags(tc.input)
			if (err != nil) != tc.invalid {
				t.Fatalf("error = %v", err)
			}
			if !tc.invalid && !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}
