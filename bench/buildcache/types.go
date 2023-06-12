package buildcache

type CacheObjectType string

var (
	BuildObj CacheObjectType = "build"
	IniObj   CacheObjectType = "INI"
)

func (ct CacheObjectType) String() string {
	switch ct {
	case BuildObj:
		return "build"
	case IniObj:
		return "ini"
	default:
		return "Unknown"
	}
}
