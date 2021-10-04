package main

import (
	"testing"
)

func TestGetEditor(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{
			"Returns VISUAL",
			"code",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetEditor()

			if (err != nil) != tt.wantErr {
				t.Errorf("GetEditor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetEditor() = %v, want %v", got, tt.want)
			}
		})
	}
}
