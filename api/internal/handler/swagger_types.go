package handler

// ── Swagger-friendly response types ──────────────────────────────────────────
// swaggo cannot resolve pgtype.* and shopspring/decimal types from the store
// package. These structs mirror the JSON output of the store types using
// primitive Go types so the generated OpenAPI spec is accurate.
//
// These types are ONLY used in @Success/@Failure annotations — the actual
// handler code still returns the real store types, which serialize identically.

// SwaggerRepair is the OpenAPI representation of store.Repair.
type SwaggerRepair struct {
	ID             string  `json:"id"               example:"2ebb4c68-4da4-4c93-bb6c-6ad7cf0e251a"`
	VehicleID      string  `json:"vehicle_id"       example:"a44d8522-66b3-4146-af64-9db60c72470a"`
	OrgID          string  `json:"org_id"           example:"00000000-0000-0000-0000-000000000003"`
	CreatedBy      string  `json:"created_by"       example:"00000000-0000-0000-0000-000000000002"`
	AssignedTo     *string `json:"assigned_to"      example:"00000000-0000-0000-0000-000000000002"`
	TemplateID     *string `json:"template_id"`
	Title          string  `json:"title"            example:"Brake pad replacement"`
	System         string  `json:"system"           example:"brakes"`
	Status         string  `json:"status"           example:"in_progress"`
	ProgressPct    int16   `json:"progress_pct"     example:"50"`
	TotalPartsCost float64 `json:"total_parts_cost" example:"34.99"`
	StartedAt      *string `json:"started_at"       example:"2026-04-10T20:00:00Z"`
	CompletedAt    *string `json:"completed_at"`
	CreatedAt      string  `json:"created_at"       example:"2026-04-10T19:00:00Z"`
	UpdatedAt      string  `json:"updated_at"       example:"2026-04-10T20:00:00Z"`
}

// SwaggerRepairStep is the OpenAPI representation of store.RepairStep.
type SwaggerRepairStep struct {
	ID          string  `json:"id"           example:"bbca9438-c5ba-411e-820b-aa659f20eb93"`
	RepairID    string  `json:"repair_id"    example:"2ebb4c68-4da4-4c93-bb6c-6ad7cf0e251a"`
	Position    int16   `json:"position"     example:"1"`
	Title       string  `json:"title"        example:"Remove wheel"`
	Notes       *string `json:"notes"        example:"Use 21mm socket"`
	TorqueSpec  *string `json:"torque_spec"  example:"100 ft-lbs"`
	Completed   bool    `json:"completed"    example:"false"`
	CompletedAt *string `json:"completed_at"`
	CreatedAt   string  `json:"created_at"   example:"2026-04-10T19:00:00Z"`
}

// SwaggerPart is the OpenAPI representation of store.Part.
type SwaggerPart struct {
	ID         string   `json:"id"          example:"bbca9438-c5ba-411e-820b-aa659f20eb93"`
	RepairID   string   `json:"repair_id"   example:"2ebb4c68-4da4-4c93-bb6c-6ad7cf0e251a"`
	Name       string   `json:"name"        example:"Brake pad set"`
	PartNumber *string  `json:"part_number" example:"D1060-8J090"`
	Supplier   *string  `json:"supplier"    example:"RockAuto"`
	Status     string   `json:"status"      example:"ordered"`
	Cost       *float64 `json:"cost"        example:"34.99"`
	Notes      *string  `json:"notes"`
	CreatedAt  string   `json:"created_at"  example:"2026-04-10T19:00:00Z"`
}

// SwaggerRepairNote is the OpenAPI representation of store.RepairNote.
type SwaggerRepairNote struct {
	ID        string `json:"id"         example:"bbca9438-c5ba-411e-820b-aa659f20eb93"`
	RepairID  string `json:"repair_id"  example:"2ebb4c68-4da4-4c93-bb6c-6ad7cf0e251a"`
	UserID    string `json:"user_id"    example:"00000000-0000-0000-0000-000000000002"`
	Body      string `json:"body"       example:"Noticed scoring on the rotor."`
	CreatedAt string `json:"created_at" example:"2026-04-10T19:00:00Z"`
}

// SwaggerVehicle is the OpenAPI representation of store.Vehicle.
type SwaggerVehicle struct {
	ID        string  `json:"id"         example:"a44d8522-66b3-4146-af64-9db60c72470a"`
	OrgID     string  `json:"org_id"     example:"00000000-0000-0000-0000-000000000003"`
	Year      int16   `json:"year"       example:"2019"`
	Make      string  `json:"make"       example:"Ford"`
	Model     string  `json:"model"      example:"F-150"`
	Trim      *string `json:"trim"       example:"XLT"`
	Vin       *string `json:"vin"`
	Mileage   *int32  `json:"mileage"    example:"84200"`
	UnitLabel *string `json:"unit_label"`
	CreatedAt string  `json:"created_at" example:"2026-04-10T19:00:00Z"`
}

// SwaggerActiveRepair is the OpenAPI representation of store.ListActiveRepairsRow.
type SwaggerActiveRepair struct {
	SwaggerRepair
	Year      int16   `json:"year"       example:"2019"`
	Make      string  `json:"make"       example:"Ford"`
	Model     string  `json:"model"      example:"F-150"`
	Trim      *string `json:"trim"       example:"XLT"`
	UnitLabel *string `json:"unit_label"`
}

// SwaggerRepairDetail is the OpenAPI representation of repairDetail.
type SwaggerRepairDetail struct {
	SwaggerRepair
	Steps []SwaggerRepairStep `json:"steps"`
	Parts []SwaggerPart       `json:"parts"`
}

// SwaggerDeletedResponse is used for delete/toggle endpoints that return {deleted, id}.
type SwaggerDeletedResponse struct {
	Deleted bool   `json:"deleted" example:"true"`
	ID      string `json:"id"      example:"bbca9438-c5ba-411e-820b-aa659f20eb93"`
}

// SwaggerStepToggleResponse is the response from the CompleteStep toggle endpoint.
type SwaggerStepToggleResponse struct {
	Step   SwaggerRepairStep `json:"step"`
	Repair SwaggerRepair     `json:"repair"`
}
