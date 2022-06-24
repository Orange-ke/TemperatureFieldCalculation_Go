package model

type Env struct {
	LevelHeight              float32                        `json:"level_height"`
	SteelValue               int                            `json:"steel_value"`
	StartTemperature         float32                        `json:"start_temperature"`
	Md                       Md                             `json:"md"`
	DragSpeed                float32                        `json:"drag_speed"`
	Coordinate               Coordinate                     `json:"coordinate"`
	SecondaryCoolingWaterCfg []SecondaryCoolingWaterSection `json:"secondary_cooling_water_cfg"`
	CoolingZoneCfg           []CoolingZone                  `json:"cooling_zone_cfg"`
}

// 铸机尺寸配置
type Coordinate struct {
	R                   float32 `json:"r"`
	LevelHeight         float32 `json:"level_height"`
	ArcStartDistance    float32 `json:"arc_start_distance"`
	ArcEndDistance      float32 `json:"arc_end_distance"`
	CenterStartDistance float32 `json:"center_start_distance"`
	CenterEndDistance   float32 `json:"center_end_distance"`
	MdLength            int     `json:"md_length"`
	Width               int     `json:"width"`
	Length              int     `json:"length"`
	ZLength             int     `json:"z_length"`
	ZScale              int     `json:"z_scale"`
	XScale              int     `json:"x_scale"`
	YScale              int     `json:"y_scale"`
}

// 冷却区分区配置
type CoolingZone struct {
	ZoneName    string  `json:"zone_name"`
	Start       int     `json:"start"`
	End         int     `json:"end"`
	Medium      int     `json:"medium"`
	EndDistance float32 `json:"end_distance"`
}

// 结晶器冷却参数
type Md struct {
	NarrowSurfaceIn     float32 `json:"narrow_surface_in"`
	NarrowSurfaceOut    float32 `json:"narrow_surface_out"`
	NarrowSurfaceVolume float32 `json:"narrow_surface_volume"`
	WideSurfaceIn       float32 `json:"wide_surface_in"`
	WideSurfaceOut      float32 `json:"wide_surface_out"`
	WideSurfaceVolume   float32 `json:"wide_surface_volume"`
}

// 结晶器窄面
type NarrowSurface struct {
	In     float32 `json:"in"`
	Out    float32 `json:"out"`
	Volume float32 `json:"volume"`
}

// 结晶器宽面
type WideSurface struct {
	In     float32 `json:"in"`
	Out    float32 `json:"out"`
	Volume float32 `json:"volume"`
}

type Speed2Water struct {
	Top    float32 `json:"top"`
	Bottom float32 `json:"bottom"`
	Step   float32 `json:"step"`
}

// 纵切面云图请求结构体
type VerticalReqData struct {
	Index  int `json:"index"`
	ZScale int `json:"z_scale"`
}

// 物性参数
type PhysicalParameter struct {
	Id                  int       `json:"id"`
	Temperature         float32   `json:"temperature"`
	ThermalConductivity float32   `json:"thermal_conductivity"`
	SpecficHeat         float32   `json:"specfic_heat"`
	Density             float32   `json:"density"`
	Enthalpy            float32   `json:"enthalpy"`
	LiquidPhaseFraction float32   `json:"liquid_phase_fraction"`
	Emissivity          float32   `json:"emissivity"`
	PoissonRatio        float32   `json:"poisson_ratio"`
	LinearExpansion     float32   `json:"linear_expansion"`
	YoungModulus        float32   `json:"young_modulus"`
	SteelType           SteelType `json:"steel_type"`
}

// 固液相线温度参数
type PhaseTemperature struct {
	Id                     int       `json:"id"`
	LiquidPhaseTemperature float32   `json:"liquid_phase_temperature"`
	SolidPhaseTemperature  float32   `json:"solid_phase_temperature"`
	SteelType              SteelType `json:"steel_type"`
}

type SteelType struct {
	Id                int               `json:"id"`
	Name              string            `json:"name"`
	SteelTypeCategory SteelTypeCategory `json:"steel_type_category"`
}

type SteelTypeCategory struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

// 铸机冷却参数配置
type CoolerCfg struct {
	StartTemperature float32 // 初始浇铸温度
	// 结晶器冷却参数配置 ---- start
	NarrowSurfaceIn   float32 // 窄面入水温度
	NarrowSurfaceOut  float32 // 窄面出水温度
	NarrowWaterVolume float32 // 窄面水量

	WideSurfaceIn   float32 // 宽面入水温度
	WideSurfaceOut  float32 // 宽面出水温度
	WideWaterVolume float32 // 宽面水量
	// 结晶器冷却参数配置 ---- end

	// todo 暂时未用到
	TargetTemperature map[string]float32 // 每个冷却区的目标温度

	V int64 // 拉速

	// 二冷区冷却参数配置
	SecondaryCoolingZoneCfg SecondaryCoolingZoneCfg
}

type SecondaryCoolingZoneCfg struct {
	SecondaryCoolingWaterCfg []SecondaryCoolingWaterSection `json:"secondary_cooling_water_cfg"`
	NozzleCfg                NozzleCfg                      `json:"nozzle_cfg"`
	CoolingZoneCfg           []CoolingZone                  `json:"cooling_zone_cfg"`
}

type SecondaryCoolingWaterSection struct {
	SprayWaterTemperature float32 `json:"spray_water_temperature"`
	InnerArcWaterVolume   float32 `json:"inner_arc_water_volume"`
	NarrowSideWaterVolume float32 `json:"narrow_side_water_volume"`
	Fuqie1Volume          float32 `json:"fuqie_1_volume"`
	Fuqie2Volume          float32 `json:"fuqie_2_volume"`
}

// 喷嘴布置配置
type NozzleCfg struct {
	WideItems   []WideItem   `json:"wide_items"`
	NarrowItems []NarrowItem `json:"narrow_items"`
}

type WideItem struct {
	RollerNum                     int     `json:"roller_num"`
	CoolingZone                   int     `json:"cooling_zone"`
	OuterDiameter                 int     `json:"outer_diameter"`
	InnerDiameter                 int     `json:"inner_diameter"`
	Medium                        int     `json:"medium"`
	Distance                      float32 `json:"distance"`
	RollerInnerDiameter           int     `json:"roller_inner_diameter"`
	RollerDistance                float32 `json:"roller_distance"`
	CenterSpraySection            Section `json:"center_spray_section"`
	AlterSpraySection1            Section `json:"alter_spray_section_1"`
	AlterSpraySection2            Section `json:"alter_spray_section_2"`
	ElectromagneticStirringFactor float32 `json:"electromagnetic_stirring_factor"`
}

type NarrowItem struct {
	RollerNum      int           `json:"roller_num"`
	CoolingZone    int           `json:"cooling_zone"`
	Diameter       float32       `json:"diameter"`
	RollerDistance float32       `json:"roller_distance"`
	SpraySection1  NarrowSection `json:"spray_section_1"`
	SpraySection2  NarrowSection `json:"spray_section_2"`
	SpraySection3  NarrowSection `json:"spray_section_3"`
}

type Section struct {
	LeftLimit  float32 `json:"left_limit"`
	RightLimit float32 `json:"right_limit"`
	Thickness  float32 `json:"thickness"`
}

type NarrowSection struct {
	Width     float32 `json:"width"`
	Thickness float32 `json:"thickness"`
}

// 前后端通信消息结构
type Msg struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

const (
	XStep  = 5
	YStep  = 5
	ZStep  = 10
	Length = 2700 / 2 // 最大宽面长度
	Width  = 420 / 2  // 最大窄面长度

	// 存放热流密度和综合换热系数容器的元素最大长度
	WL = Length/XStep + Width/YStep
)

// 元素类型
type ItemType [Width / YStep][Length / XStep]float32
