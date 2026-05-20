package deps

import "testing"

func TestLogIndicatesStalePodSpecs(t *testing.T) {
	tests := []struct {
		name string
		log  string
		want bool
	}{
		{
			name: "empty log",
			log:  "",
			want: false,
		},
		{
			name: "unrelated failure",
			log:  "Error: unable to find a target named `App`",
			want: false,
		},
		{
			name: "missing spec dependency",
			log:  "[!] None of your spec sources contain a spec satisfying the dependency: `Foo`.",
			want: true,
		},
		{
			name: "out-of-date source repos",
			log:  "CocoaPods could not find compatible versions; your out-of-date source repos may be the cause.",
			want: true,
		},
		{
			name: "case insensitive match",
			log:  "NONE OF YOUR SPEC SOURCES CONTAIN A SPEC SATISFYING THE DEPENDENCY",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := logIndicatesStalePodSpecs(tt.log); got != tt.want {
				t.Errorf("logIndicatesStalePodSpecs() = %v, want %v", got, tt.want)
			}
		})
	}
}
