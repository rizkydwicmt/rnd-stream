package common

import "time"

type Ticket struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	TicketNo    string    `json:"ticket_no"`
	CustomerID  uint      `json:"customer_id"`
	Subject     string    `json:"subject"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Ticket) TableName() string {
	return "tickets"
}
