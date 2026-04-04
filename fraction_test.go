package recipemd

import "testing"

func TestFormatFactorFraction(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		factor float64
		want   string
	}{
		{name: "whole number", factor: 3, want: "3"},
		{name: "half", factor: 0.5, want: "½"},
		{name: "quarter", factor: 0.25, want: "¼"},
		{name: "three quarters", factor: 0.75, want: "¾"},
		{name: "one third", factor: 1.0 / 3.0, want: "⅓"},
		{name: "two thirds", factor: 2.0 / 3.0, want: "⅔"},
		{name: "one and a half", factor: 1.5, want: "1½"},
		{name: "three and a quarter", factor: 3.25, want: "3¼"},
		{name: "two and two thirds", factor: 2.0 + 2.0/3.0, want: "2⅔"},
		{name: "one eighth", factor: 0.125, want: "⅛"},
		{name: "seven eighths", factor: 0.875, want: "⅞"},
		{name: "zero", factor: 0, want: "0"},
		{name: "negative half", factor: -0.5, want: "-½"},
		{name: "negative one and half", factor: -1.5, want: "-1½"},
		// Fallback to decimal for non-standard fractions
		{name: "decimal fallback", factor: 1.17, want: "1.17"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := Amount{Factor: tt.factor}
			got := a.FormatFactorFraction()
			if got != tt.want {
				t.Errorf("FormatFactorFraction(%f) = %q, want %q", tt.factor, got, tt.want)
			}
		})
	}
}

func TestSerializeFraction(t *testing.T) {
	t.Parallel()
	cup := "cup"
	tests := []struct {
		name   string
		amount Amount
		want   string
	}{
		{name: "with unit", amount: Amount{Factor: 1.5, Unit: &cup}, want: "1½ cup"},
		{name: "no unit", amount: Amount{Factor: 0.25}, want: "¼"},
		{name: "whole with unit", amount: Amount{Factor: 3, Unit: &cup}, want: "3 cup"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.amount.SerializeFraction()
			if got != tt.want {
				t.Errorf("SerializeFraction() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSerializeCompound(t *testing.T) {
	t.Parallel()
	cup := "cup"
	tbsp := "tbsp"
	amounts := []Amount{
		{Factor: 1, Unit: &cup},
		{Factor: 1, Unit: &tbsp},
	}
	got := SerializeCompound(amounts)
	want := "1 cup and 1 tbsp"
	if got != want {
		t.Errorf("SerializeCompound() = %q, want %q", got, want)
	}
}
