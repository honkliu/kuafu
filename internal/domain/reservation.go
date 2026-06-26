package domain

import "time"

// ReservationStatus represents the lifecycle state of a node reservation.
type ReservationStatus string

const (
	ReservationStatusActive   ReservationStatus = "Active"
	ReservationStatusReleased ReservationStatus = "Released"
)

// ReservationCommandStatus represents the state of a command attached to a reservation.
type ReservationCommandStatus string

const (
	ReservationCommandStatusRecorded ReservationCommandStatus = "Recorded"
)

// NodeReservation reserves one or more nodes for direct interactive/container work.
type NodeReservation struct {
	ID        string               `json:"id"`
	Name      string               `json:"name"`
	Owner     string               `json:"owner,omitempty"`
	NodeNames []string             `json:"nodeNames"`
	Status    ReservationStatus    `json:"status"`
	CreatedAt time.Time            `json:"createdAt"`
	ExpiresAt time.Time            `json:"expiresAt,omitempty"`
	Commands  []ReservationCommand `json:"commands,omitempty"`
}

// ReservationCreateRequest is the API payload for creating a node reservation.
type ReservationCreateRequest struct {
	Name          string   `json:"name"`
	Owner         string   `json:"owner,omitempty"`
	NodeNames     []string `json:"nodeNames"`
	DurationHours int      `json:"durationHours,omitempty"`
}

// ReservationCommandRequest is the API payload for preparing a command on reserved nodes.
type ReservationCommandRequest struct {
	Image            string                   `json:"image"`
	DockerRunOptions string                   `json:"dockerRunOptions,omitempty"`
	EntryPoint       string                   `json:"entryPoint,omitempty"`
	StartCommand     string                   `json:"startCommand"`
	WorkingDirectory string                   `json:"workingDirectory,omitempty"`
	Environment      []string                 `json:"environment,omitempty"`
	RenderedCommands []ReservationNodeCommand `json:"renderedCommands,omitempty"`
}

// ReservationCommand records the per-node command materialized from a request.
type ReservationCommand struct {
	ID               string                   `json:"id"`
	Image            string                   `json:"image"`
	DockerRunOptions string                   `json:"dockerRunOptions,omitempty"`
	EntryPoint       string                   `json:"entryPoint,omitempty"`
	StartCommand     string                   `json:"startCommand"`
	WorkingDirectory string                   `json:"workingDirectory,omitempty"`
	Environment      []string                 `json:"environment,omitempty"`
	RenderedCommands []ReservationNodeCommand `json:"renderedCommands"`
	Status           ReservationCommandStatus `json:"status"`
	CreatedAt        time.Time                `json:"createdAt"`
	Logs             []string                 `json:"logs,omitempty"`
}

// ReservationNodeCommand is the exact command Kuafu will run on one reserved node.
type ReservationNodeCommand struct {
	NodeName string `json:"nodeName"`
	Command  string `json:"command"`
}
