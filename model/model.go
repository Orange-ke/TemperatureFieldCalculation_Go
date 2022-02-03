package model

type Env struct {
	// MachineValue int `json:"machine_value"`
	SteelValue             int         `json:"steel_value"`
	StartTemperature       float32     `json:"start_temperature"`
	NarrowSurfaceIn        float32     `json:"narrow_surface_in"`
	NarrowSurfaceOut       float32     `json:"narrow_surface_out"`
	WideSurfaceIn          float32     `json:"wide_surface_in"`
	WideSurfaceOut         float32     `json:"wide_surface_out"`
	SprayTemperature       float32     `json:"spray_temperature"`
	RollerWaterTemperature float32     `json:"roller_water_temperature"`
	DragSpeed              float32     `json:"drag_speed"`
	CalculateMethodValue   int         `json:"calculate_method_value"`
	Speed2Water            Speed2Water `json:"speed_2_water"`
}

type Speed2Water struct {
	Top    float32 `json:"top"`
	Bottom float32 `json:"bottom"`
	Step   float32 `json:"step"`
}

type Msg struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}
