package parse

import (
	"github.com/google/go-cmp/cmp"
	"strings"
	"testing"
)

func anyError(t *testing.T, err error) {

}

func TestParse(t *testing.T) {
	t.Parallel()

	type tcase struct {
		name    string
		body    string
		want    Result
		wantErr func(t *testing.T, err error)
	}
	for _, tc := range []tcase{
		{
			name: "Simple1Recipe",
			body: `
wget:
  install: "apt-get install -y wget"
  check: "wget -h"
`,
			want: Result{
				Recipes: []Recipe{
					{
						Name: "wget",
						Installers: []Installer{
							{
								Script: "apt-get install -y wget",
							},
						},
						Check: "wget -h",
					},
				},
			},
		},
		{
			name: "OverloadedTarget",
			body: `
wget:
  check: "wget -h"
  install_apt:
    script: "apt install wget"
    deps:
      - apt
  install_brew:
    script: "brew install wget"
    deps:
      - brew
`,
			want: Result{
				Recipes: []Recipe{
					{
						Name: "wget",
						Installers: []Installer{
							{
								Name:         "apt",
								Script:       "apt install wget",
								Dependencies: []string{"apt"},
							},
							{
								Name:         "brew",
								Script:       "brew install wget",
								Dependencies: []string{"brew"},
							},
						},
						Check: "wget -h",
					},
				},
			},
		},
		{
			name: "Empty",
			body: `
`,
		},
		{
			name: "BrokenRecipe",
			body: `
curl:
  dog: "dog"
`,
			wantErr: anyError,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("parsing %s", tc.body)
			res, err := Parse(
				strings.NewReader(strings.TrimSpace(tc.body)),
			)
			if err != nil {
				if tc.wantErr == nil {
					t.Fatalf("unexpected error: %v", err)
				}
				tc.wantErr(t, err)
				return
			}

			if !cmp.Equal(*res, tc.want) {
				t.Error(cmp.Diff(*res, tc.want))
			}
		})
	}
}
