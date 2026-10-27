package ledger

import (
	"context"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
)

// AssignCard links a card's last four digits to an account, moving it from any other.
func (s *Service) AssignCard(ctx context.Context, userID, accountID int64, mask string) error {
	return s.inTx(ctx, func(q *dbq.Queries) error {
		if _, err := s.ownAccount(ctx, q, userID, accountID, "account_id"); err != nil {
			return err
		}
		if err := q.ReleaseCardMask(ctx, dbq.ReleaseCardMaskParams{UserID: userID, CardMask: &mask}); err != nil {
			return err
		}
		return q.SetCardMask(ctx, dbq.SetCardMaskParams{CardMask: &mask, ID: accountID, UserID: userID})
	})
}

// Adjust changes an account's balance by delta without counting it as income or spending:
// a balance correction, or money from an account the user doesn't track.
func (s *Service) Adjust(ctx context.Context, userID, accountID, delta int64, on time.Time, description string) (Transaction, error) {
	var out Transaction
	if delta == 0 {
		return out, invalid("invalid_amount", "amount_minor", "nothing to adjust")
	}
	err := s.inTx(ctx, func(q *dbq.Queries) error {
		a, err := s.ownAccount(ctx, q, userID, accountID, "account_id")
		if err != nil {
			return err
		}
		tx, err := q.InsertTransaction(ctx, dbq.InsertTransactionParams{
			UserID: userID, Type: string(Opening), OccurredOn: on, Description: description, Source: "manual",
		})
		if err != nil {
			return err
		}
		if err := writeLegs(ctx, q, userID, tx.ID, legsFor(Opening, delta, a, nil, 0)); err != nil {
			return err
		}
		if out, err = load(ctx, q, tx); err != nil {
			return err
		}
		return audit(ctx, q, userID, "account", accountID, "adjust", nil, out)
	})
	return out, err
}

// Reconcile sets an account's balance to target (for example, what the bank reported).
func (s *Service) Reconcile(ctx context.Context, userID, accountID, target int64, on time.Time, description string) (Transaction, error) {
	q := dbq.New(s.pool)
	if _, err := q.GetAssetAccount(ctx, dbq.GetAssetAccountParams{ID: accountID, UserID: userID}); err != nil {
		return Transaction{}, notFound(err)
	}
	bal, err := q.AccountBalance(ctx, accountID)
	if err != nil {
		return Transaction{}, err
	}
	return s.Adjust(ctx, userID, accountID, target-bal, on, description)
}

// Balance returns an account's current balance.
func (s *Service) Balance(ctx context.Context, accountID int64) (int64, error) {
	return dbq.New(s.pool).AccountBalance(ctx, accountID)
}
