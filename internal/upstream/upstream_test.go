package upstream

import "testing"

// TestParamNames 上游查询参数名是拼查询串与进缓存键的依据。
//
// 值随上游文档走，改了要让所有用它的端点跟着改。
func TestParamNames(t *testing.T) {
	want := map[string]string{
		"Lang":       "lang",
		"LocalTime":  "localTime",
		"Number":     "number",
		"Radius":     "radius",
		"Adm":        "adm",
		"Range":      "range",
		"City":       "city",
		"Interval":   "interval",
		"Tilt":       "tilt",
		"Azimuth":    "azimuth",
		"Extra":      "extra",
		"Project":    "project",
		"Credential": "credential",
		"Location":   "location",
		"Date":       "date",
		"Type":       "type",
		"Hours":      "hours",
		"Days":       "days",
		"Time":       "time",
		"Timezone":   "tz",
		"Altitude":   "alt",
		"Basin":      "basin",
		"Year":       "year",
		"StormID":    "stormid",
	}
	got := map[string]string{
		"Lang": Lang, "LocalTime": LocalTime, "Number": Number, "Radius": Radius,
		"Adm": Adm, "Range": Range, "City": City, "Interval": Interval,
		"Tilt": Tilt, "Azimuth": Azimuth, "Extra": Extra, "Project": Project,
		"Credential": Credential, "Location": Location, "Date": Date, "Type": Type,
		"Hours": Hours, "Days": Days, "Time": Time, "Timezone": Timezone,
		"Altitude": Altitude, "Basin": Basin, "Year": Year, "StormID": StormID,
	}
	if len(got) != len(want) {
		t.Fatalf("参数名 = %d, want %d", len(got), len(want))
	}
	for name, w := range want {
		if got[name] != w {
			t.Errorf("%s = %q, want %q", name, got[name], w)
		}
	}
}
