package qwx

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/KanoCifer/qwx/internal/coord"
)

// Location 表示一个查询位置。
//
// 和风天气的位置有三种形态，本包统一收敛到这一个类型：
//
//   - 坐标对："116.41,39.92"（**经度在前**）
//   - LocationID：如 "101010100"（行政区划）、"10101010012A"（POI）、
//     "P2352"（潮汐站）
//   - 地名：仅 GeoAPI 认
type Location struct {
	// ID 是 LocationID，如 "101010100"。与 Lat/Lon 二选一。
	ID string
	// Name 是地名，如 "北京"。仅 GeoAPI 的城市/POI 检索认。
	Name string
	// Lat 是纬度。
	Lat float64
	// Lon 是经度。
	Lon float64
}

// locationIDRe LocationID 的形态。
//
// 和风天气的 LocationID 有好几种长相：行政区划是 9~10 位纯数字
// （101010100）、POI 是数字加字母后缀（10101010012A）、潮汐站点是
// 字母 P 打头（P2352）。三种都要放行。
//
// 不含小数点：裸经度（"121.47"）三种形态都不沾，上游直接回 400。
var locationIDRe = regexp.MustCompile(`^[A-Za-z]?[0-9]{4,}[A-Za-z]?$`)

// ParseLocation 解析 location 字符串，识别坐标对与 LocationID。
//
// 识别不了形态时返回 [ErrInvalidLocation]。
func ParseLocation(raw string) (Location, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return Location{}, fmt.Errorf("%w: empty location", ErrInvalidLocation)
	}
	if strings.Contains(s, ",") {
		lat, lon, err := parseCoordinatePair(s)
		if err != nil {
			return Location{}, err
		}
		return Location{Lat: lat, Lon: lon}, nil
	}
	if locationIDRe.MatchString(s) {
		return Location{ID: s}, nil
	}
	return Location{}, fmt.Errorf("%w: %q is neither a coordinate pair nor a LocationID", ErrInvalidLocation, raw)
}

// parseCoordinatePair 解析 "经度,纬度"，并做范围校验。
func parseCoordinatePair(s string) (lat, lon float64, err error) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("%w: %q is not a coordinate pair", ErrInvalidLocation, s)
	}
	lon, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: %q has a non-numeric longitude", ErrInvalidLocation, s)
	}
	lat, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: %q has a non-numeric latitude", ErrInvalidLocation, s)
	}
	// ParseFloat 接受 "NaN" / "Inf"，它们能过范围比较（NaN 的任何比较都是
	// false），落到上游就是又一次 400，这里显式挡掉。
	if math.IsNaN(lon) || math.IsNaN(lat) || math.IsInf(lon, 0) || math.IsInf(lat, 0) {
		return 0, 0, fmt.Errorf("%w: %q contains NaN or Inf", ErrInvalidLocation, s)
	}
	if lon < -180 || lon > 180 {
		return 0, 0, fmt.Errorf("%w: longitude %v out of range [-180,180]", ErrInvalidLocation, lon)
	}
	if lat < -90 || lat > 90 {
		return 0, 0, fmt.Errorf("%w: latitude %v out of range [-90,90]", ErrInvalidLocation, lat)
	}
	return lat, lon, nil
}

// HasCoordinates 报告该位置是否带坐标。
func (l Location) HasCoordinates() bool {
	return l.Lat != 0 || l.Lon != 0
}

// IsZero 报告该位置是否为空。
func (l Location) IsZero() bool {
	return l.ID == "" && l.Name == "" && !l.HasCoordinates()
}

// Query 返回该位置在 v7/GeoAPI 的 location 查询参数形态。
//
// 优先用坐标（"经度,纬度"），其次 LocationID，最后地名。
// 都没有时返回空串。
func (l Location) Query() string {
	switch {
	case l.HasCoordinates():
		return coord.Format(l.Lon) + "," + coord.Format(l.Lat)
	case l.ID != "":
		return l.ID
	default:
		return l.Name
	}
}

// Coordinates 返回归一化后的坐标（保留两位小数）。
func (l Location) Coordinates() (lat, lon float64, err error) {
	if !l.HasCoordinates() {
		return 0, 0, fmt.Errorf("%w: location has no coordinates", ErrInvalidLocation)
	}
	return coord.Round(l.Lat), coord.Round(l.Lon), nil
}

// Path 返回新一代接口路径里的坐标分量：**纬度在前、经度在后**。
//
// 这是和风天气接入最容易踩的坑：旧接口用 "经度,纬度" 查询参数，
// 新接口是 {latitude}/{longitude} 路径参数。写反了上游回 400
// invalidParams:["longitude","latitude"]，不会静默返回别处。
//
// 本方法只服务新一代（/weather/v1/*、/airquality/v1/*、/weatheralert/v1/*）。
// 旧一代与 GeoAPI 用 [Location.Query]，那个形态是经度在前。
func (l Location) Path() (latPart, lonPart string, err error) {
	if !l.HasCoordinates() {
		return "", "", fmt.Errorf("%w: location has no coordinates", ErrInvalidLocation)
	}
	lat, lon, err := l.Coordinates()
	if err != nil {
		return "", "", err
	}
	return coord.Format(lat), coord.Format(lon), nil
}
