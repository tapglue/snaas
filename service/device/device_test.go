package device

import "testing"

func TestValidate(t *testing.T) {
	var (
		d  = testDevice()
		ds = List{
			{}, // Missing DeviceID
			{DeviceID: d.DeviceID},                                                                  // Missing Language
			{DeviceID: d.DeviceID, Language: DefaultLanguage},                                       // Missing Platform
			{DeviceID: d.DeviceID, Language: DefaultLanguage, Platform: 4},                          // Unsupported Platform
			{DeviceID: d.DeviceID, Language: DefaultLanguage, Platform: d.Platform},                 // Missing Token
			{DeviceID: d.DeviceID, Language: DefaultLanguage, Platform: d.Platform, Token: d.Token}, // Missing UserID
		}
	)

	for _, d := range ds {
		if have, want := d.Validate(), ErrInvalidDevice; !IsInvalidDevice(have) {
			t.Errorf("have %v, want %v", have, want)
		}
	}
}
