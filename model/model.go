package model

type Env struct {
	LevelHeight      float32    `json:"level_height"`
	SteelValue       int        `json:"steel_value"`
	StartTemperature float32    `json:"start_temperature"`
	Md               Md         `json:"md"`
	DragSpeed        float32    `json:"drag_speed"`
	Coordinate       Coordinate `json:"coordinate"`
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
	WL = Length/XStep + Width/YStep - 1
)

// 元素类型
type ItemType [Width / YStep][Length / XStep]float32
