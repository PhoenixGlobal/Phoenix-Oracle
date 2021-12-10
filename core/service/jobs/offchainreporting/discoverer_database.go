package offchainreporting

import (
	"context"
	"database/sql"

	ocrnetworking "PhoenixOracle/lib/libocr/networking"
	"github.com/lib/pq"
	p2ppeer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

var _ ocrnetworking.DiscovererDatabase = &DiscovererDatabase{}

type DiscovererDatabase struct {
	db     *sql.DB
	peerID string
}

func NewDiscovererDatabase(db *sql.DB, peerID p2ppeer.ID) *DiscovererDatabase {
	return &DiscovererDatabase{
		db,
		peerID.Pretty(),
	}
}

func (d *DiscovererDatabase) StoreAnnouncement(ctx context.Context, peerID string, ann []byte) error {
	_, err := d.db.ExecContext(ctx, `
INSERT INTO offchainreporting_discoverer_announcements (local_peer_id, remote_peer_id, ann, created_at, updated_at)
VALUES ($1,$2,$3,NOW(),NOW()) ON CONFLICT (local_peer_id, remote_peer_id) DO UPDATE SET 
ann = EXCLUDED.ann,
updated_at = EXCLUDED.updated_at
;`, d.peerID, peerID, ann)
	return errors.Wrap(err, "DiscovererDatabase failed to StoreAnnouncement")
}

func (d *DiscovererDatabase) ReadAnnouncements(ctx context.Context, peerIDs []string) (map[string][]byte, error) {
	rows, err := d.db.QueryContext(ctx, `
SELECT remote_peer_id, ann FROM offchainreporting_discoverer_announcements WHERE remote_peer_id = ANY($1) AND local_peer_id = $2`, pq.Array(peerIDs), d.peerID)
	if err != nil {
		return nil, errors.Wrap(err, "DiscovererDatabase failed to ReadAnnouncements")
	}
	results := make(map[string][]byte)
	for rows.Next() {
		var peerID string
		var ann []byte
		err := rows.Scan(&peerID, &ann)
		if err != nil {
			return nil, multierr.Combine(err, rows.Close())
		}
		results[peerID] = ann
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, errors.WithStack(err)
	}
	return results, nil
}
