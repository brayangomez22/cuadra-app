package httpadapter

import (
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/adapters/http/receivablesapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

func date(d domain.Date) openapi_types.Date { return openapi_types.Date{Time: d.Time()} }

// creditBody is a customer's account with its open receivables.
func creditBody(a *domain.Account, open []*domain.Receivable) receivablesapi.CustomerCredit {
	items := make([]receivablesapi.Receivable, 0, len(open))
	for _, r := range open {
		items = append(items, receivableBody(r))
	}
	return receivablesapi.CustomerCredit{
		CustomerId: a.CustomerID(), CreditLimit: httpx.FormatMoney(a.Terms().Limit()), TermDays: a.Terms().Days(),
		Balance: httpx.FormatMoney(a.Balance()), Available: httpx.FormatMoney(a.Available()), Receivables: items,
	}
}

func receivableBody(r *domain.Receivable) receivablesapi.Receivable {
	body := receivablesapi.Receivable{
		Id: r.ID(), SaleId: r.SaleID(), Amount: httpx.FormatMoney(r.Amount()), Balance: httpx.FormatMoney(r.Balance()),
		DueDate: date(r.DueDate()), Status: receivablesapi.ReceivableStatus(r.Status()), CreatedAt: r.CreatedAt(),
	}
	if auth := r.Authorization(); auth != nil {
		body.AuthorizedBy = &auth.UserID
	}
	return body
}

func movementBody(m domain.Movement) receivablesapi.CreditMovement {
	body := receivablesapi.CreditMovement{
		Kind: receivablesapi.CreditMovementKind(m.Kind), At: m.At,
		Amount: httpx.FormatMoney(m.Amount), Balance: httpx.FormatMoney(m.Balance),
		SaleId: m.SaleID, PaymentId: m.PaymentID,
	}
	if m.Method != "" {
		method := receivablesapi.CustomerPaymentMethod(m.Method)
		body.Method = &method
	}
	if m.DueDate != nil {
		d := date(*m.DueDate)
		body.DueDate = &d
	}
	return body
}

func paymentBody(res app.PaymentResult) receivablesapi.CustomerPayment {
	p := res.Payment
	allocations := make([]receivablesapi.PaymentAllocation, 0, len(p.Allocations()))
	for _, a := range p.Allocations() {
		allocations = append(allocations, receivablesapi.PaymentAllocation{ReceivableId: a.ReceivableID, Amount: httpx.FormatMoney(a.Amount)})
	}
	return receivablesapi.CustomerPayment{
		Id: p.ID(), CustomerId: p.CustomerID(), Method: receivablesapi.CustomerPaymentMethod(p.Method()),
		Amount: httpx.FormatMoney(p.Amount()), CashSessionId: p.CashSessionID(), CreatedAt: p.CreatedAt(),
		Allocations: allocations, Balance: httpx.FormatMoney(res.Account.Balance()),
	}
}

func bucketsBody(g *domain.Aging) receivablesapi.AgingBuckets {
	return receivablesapi.AgingBuckets{
		Current: httpx.FormatMoney(g.Current), Days130: httpx.FormatMoney(g.Days1To30),
		Days3160: httpx.FormatMoney(g.Days31To60), Days6190: httpx.FormatMoney(g.Days61To90),
		Over90: httpx.FormatMoney(g.Over90), Overdue: httpx.FormatMoney(g.Overdue()), Total: httpx.FormatMoney(g.Total()),
	}
}
