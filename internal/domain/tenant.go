package domain

import "time"

type ProjectRole string

const (
	ProjectRoleAdmin     ProjectRole = "Admin"
	ProjectRoleSubmitter ProjectRole = "Submitter"
	ProjectRoleViewer    ProjectRole = "Viewer"
)

type ProjectMembership struct {
	Project string      `json:"project"`
	Role    ProjectRole `json:"role"`
}

type User struct {
	Alias       string              `json:"alias"`
	Email       string              `json:"email,omitempty"`
	DisplayName string              `json:"displayName,omitempty"`
	Memberships []ProjectMembership `json:"memberships,omitempty"`
	CreatedAt   time.Time           `json:"createdAt"`
}

func (u *User) RoleForProject(project string) (ProjectRole, bool) {
	if u == nil {
		return "", false
	}
	for _, membership := range u.Memberships {
		if membership.Project == project {
			return membership.Role, true
		}
	}
	return "", false
}

type Project struct {
	Name         string    `json:"name"`
	DisplayName  string    `json:"displayName,omitempty"`
	DefaultQueue string    `json:"defaultQueue,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}
