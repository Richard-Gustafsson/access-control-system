package models

type Door struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Location        string `json:"location"`
	MinRoleRequired string `json:"min_role_required"`
}
