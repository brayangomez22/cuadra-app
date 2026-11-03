package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const maxVoidReasonLength = 500

// Voiding is who voided a purchase, when and why.
type Voiding struct {
	By     uuid.UUID
	Reason string
	At     time.Time
}

func (v *Voiding) valid() bool {
	return v.By != uuid.Nil && v.Reason != "" && !v.At.IsZero()
}

func (v *Voiding) clone() *Voiding {
	if v == nil {
		return nil
	}
	c := *v
	return &c
}

// Void voids a purchase registered by mistake: it owes nothing from then on.
// Its stock is taken out by the use case, in the same transaction. A
// purchase on credit that already has payments to the supplier is not
// voided: the payments would be left without a purchase. The reason is
// required.
func (p *Purchase) Void(userID uuid.UUID, reason string, now time.Time) error {
	if p.voided != nil {
		return ErrPurchaseAlreadyVoided
	}
	if p.payment == PaymentCredit && p.balanceDue.LessThan(p.totals.Total) {
		return ErrPurchaseHasPayments
	}
	if userID == uuid.Nil {
		return ErrInvalidUserID
	}
	reason, err := requiredText(reason, maxVoidReasonLength, ErrInvalidVoidReason)
	if err != nil {
		return err
	}
	p.voided = &Voiding{By: userID, Reason: reason, At: now.UTC()}
	p.balanceDue = decimal.Zero
	return nil
}

// Voided returns a copy of who voided the purchase, when and why, or nil.
func (p *Purchase) Voided() *Voiding { return p.voided.clone() }
