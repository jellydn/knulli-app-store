package sdlui

import "testing"

func TestParseInputMode(t *testing.T) {
	for _, test := range []struct {
		value   string
		want    InputMode
		wantErr bool
	}{
		{value: "", want: InputAuto},
		{value: "auto", want: InputAuto},
		{value: "keyboard", want: InputKeyboard},
		{value: "controller", wantErr: true},
	} {
		t.Run(test.value, func(t *testing.T) {
			got, err := ParseInputMode(test.value)
			if (err != nil) != test.wantErr || got != test.want {
				t.Fatalf("ParseInputMode(%q) = %q, %v; want %q, error=%v", test.value, got, err, test.want, test.wantErr)
			}
		})
	}
}
