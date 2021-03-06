package db

import (
	"database/sql"
	"path"
	"sync"

	logging "gx/ipfs/QmZChCsSt8DctjceaL56Eibc29CVQq4dGKRXC5JRZ6Ppae/go-log"

	_ "github.com/mutecomm/go-sqlcipher"
	"github.com/textileio/textile-go/repo"
)

var log = logging.Logger("tex-datastore")

type SQLiteDatastore struct {
	config             repo.ConfigStore
	contacts           repo.ContactStore
	files              repo.FileStore
	threads            repo.ThreadStore
	threadInvites      repo.ThreadInviteStore
	threadPeers        repo.ThreadPeerStore
	threadMessages     repo.ThreadMessageStore
	blocks             repo.BlockStore
	notifications      repo.NotificationStore
	cafeSessions       repo.CafeSessionStore
	cafeRequests       repo.CafeRequestStore
	cafeMessages       repo.CafeMessageStore
	cafeClientNonces   repo.CafeClientNonceStore
	cafeClients        repo.CafeClientStore
	cafeTokens         repo.CafeTokenStore
	cafeClientThreads  repo.CafeClientThreadStore
	cafeClientMessages repo.CafeClientMessageStore
	db                 *sql.DB
	lock               *sync.Mutex
}

func Create(repoPath, pin string) (*SQLiteDatastore, error) {
	var dbPath string
	dbPath = path.Join(repoPath, "datastore", "mainnet.db")
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	if pin != "" {
		p := "pragma key='" + pin + "';"
		conn.Exec(p)
	}
	mux := new(sync.Mutex)
	sqliteDB := &SQLiteDatastore{
		config:             NewConfigStore(conn, mux, dbPath),
		contacts:           NewContactStore(conn, mux),
		files:              NewFileStore(conn, mux),
		threads:            NewThreadStore(conn, mux),
		threadInvites:      NewThreadInviteStore(conn, mux),
		threadPeers:        NewThreadPeerStore(conn, mux),
		threadMessages:     NewThreadMessageStore(conn, mux),
		blocks:             NewBlockStore(conn, mux),
		notifications:      NewNotificationStore(conn, mux),
		cafeSessions:       NewCafeSessionStore(conn, mux),
		cafeRequests:       NewCafeRequestStore(conn, mux),
		cafeMessages:       NewCafeMessageStore(conn, mux),
		cafeClientNonces:   NewCafeClientNonceStore(conn, mux),
		cafeClients:        NewCafeClientStore(conn, mux),
		cafeTokens:         NewCafeTokenStore(conn, mux),
		cafeClientThreads:  NewCafeClientThreadStore(conn, mux),
		cafeClientMessages: NewCafeClientMessageStore(conn, mux),
		db:                 conn,
		lock:               mux,
	}

	return sqliteDB, nil
}

func (d *SQLiteDatastore) Ping() error {
	return d.db.Ping()
}

func (d *SQLiteDatastore) Close() {
	d.db.Close()
}

func (d *SQLiteDatastore) Config() repo.ConfigStore {
	return d.config
}

func (d *SQLiteDatastore) Contacts() repo.ContactStore {
	return d.contacts
}

func (d *SQLiteDatastore) Files() repo.FileStore {
	return d.files
}

func (d *SQLiteDatastore) Threads() repo.ThreadStore {
	return d.threads
}

func (d *SQLiteDatastore) ThreadInvites() repo.ThreadInviteStore {
	return d.threadInvites
}

func (d *SQLiteDatastore) ThreadPeers() repo.ThreadPeerStore {
	return d.threadPeers
}

func (d *SQLiteDatastore) ThreadMessages() repo.ThreadMessageStore {
	return d.threadMessages
}

func (d *SQLiteDatastore) Blocks() repo.BlockStore {
	return d.blocks
}

func (d *SQLiteDatastore) Notifications() repo.NotificationStore {
	return d.notifications
}

func (d *SQLiteDatastore) CafeSessions() repo.CafeSessionStore {
	return d.cafeSessions
}

func (d *SQLiteDatastore) CafeRequests() repo.CafeRequestStore {
	return d.cafeRequests
}

func (d *SQLiteDatastore) CafeMessages() repo.CafeMessageStore {
	return d.cafeMessages
}

func (d *SQLiteDatastore) CafeClientNonces() repo.CafeClientNonceStore {
	return d.cafeClientNonces
}

func (d *SQLiteDatastore) CafeClients() repo.CafeClientStore {
	return d.cafeClients
}

func (d *SQLiteDatastore) CafeTokens() repo.CafeTokenStore {
	return d.cafeTokens
}

func (d *SQLiteDatastore) CafeClientThreads() repo.CafeClientThreadStore {
	return d.cafeClientThreads
}

func (d *SQLiteDatastore) CafeClientMessages() repo.CafeClientMessageStore {
	return d.cafeClientMessages
}

func (d *SQLiteDatastore) Copy(dbPath string, pin string) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	var cp string
	stmt := "select name from sqlite_master where type='table'"
	rows, err := d.db.Query(stmt)
	if err != nil {
		log.Errorf("error in copy: %s", err)
		return err
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		tables = append(tables, name)
	}
	if pin == "" {
		cp = `attach database '` + dbPath + `' as plaintext key '';`
		for _, name := range tables {
			cp = cp + "insert into plaintext." + name + " select * from main." + name + ";"
		}
	} else {
		cp = `attach database '` + dbPath + `' as encrypted key '` + pin + `';`
		for _, name := range tables {
			cp = cp + "insert into encrypted." + name + " select * from main." + name + ";"
		}
	}
	_, err = d.db.Exec(cp)
	if err != nil {
		return err
	}
	return nil
}

func (d *SQLiteDatastore) InitTables(pin string) error {
	return initDatabaseTables(d.db, pin)
}

func initDatabaseTables(db *sql.DB, pin string) error {
	var sqlStmt string
	if pin != "" {
		sqlStmt = "PRAGMA key = '" + pin + "';"
	}
	sqlStmt += `
    create table config (key text primary key not null, value blob);

    create table contacts (id text primary key not null, address text not null, username text not null, avatar text not null, inboxes blob not null, created integer not null, updated integer not null);
    create index contact_address on contacts (address);
    create index contact_username on contacts (username);
    create index contact_updated on contacts (updated);

    create table files (mill text not null, checksum text not null, source text not null, opts text not null, hash text not null, key text not null, media text not null, name text not null, size integer not null, added integer not null, meta blob, targets text, primary key (mill, checksum));
    create index file_hash on files (hash);
    create unique index file_mill_source_opts on files (mill, source, opts);

    create table threads (id text primary key not null, key text not null, sk blob not null, name text not null, schema text not null, initiator text not null, type integer not null, state integer not null, head text not null, members text not null, sharing integer not null);
    create unique index thread_key on threads (key);

    create table thread_invites (id text primary key not null, block blob not null, name text not null, contact blob not null, date integer not null);
    create index thread_invite_date on thread_invites (date);

    create table thread_peers (id text not null, threadId text not null, welcomed integer not null, primary key (id, threadId));
    create index thread_peer_id on thread_peers (id);
    create index thread_peer_threadId on thread_peers (threadId);
    create index thread_peer_welcomed on thread_peers (welcomed);

    create table blocks (id text primary key not null, threadId text not null, authorId text not null, type integer not null, date integer not null, parents text not null, target text not null, body text not null);
    create index block_threadId on blocks (threadId);
    create index block_type on blocks (type);
    create index block_date on blocks (date);
    create index block_target on blocks (target);

    create table thread_messages (id text primary key not null, peerId text not null, envelope blob not null, date integer not null);
    create index thread_message_date on thread_messages (date);

    create table notifications (id text primary key not null, date integer not null, actorId text not null, subject text not null, subjectId text not null, blockId text, target text, type integer not null, body text not null, read integer not null);
    create index notification_date on notifications (date);
    create index notification_actorId on notifications (actorId);
    create index notification_subjectId on notifications (subjectId);
    create index notification_blockId on notifications (blockId);
    create index notification_read on notifications (read);

    create table cafe_sessions (cafeId text primary key not null, access text not null, refresh text not null, expiry integer not null, cafe blob not null);

    create table cafe_requests (id text primary key not null, peerId text not null, targetId text not null, cafeId text not null, cafe blob not null, type integer not null, date integer not null);
    create index cafe_request_cafeId on cafe_requests (cafeId);
    create index cafe_request_date on cafe_requests (date);

    create table cafe_messages (id text primary key not null, peerId text not null, date integer not null, attempts integer not null);
    create index cafe_message_date on cafe_messages (date);

    create table cafe_client_nonces (value text primary key not null, address text not null, date integer not null);

    create table cafe_clients (id text primary key not null, address text not null, created integer not null, lastSeen integer not null, tokenId text not null);
    create index cafe_client_address on cafe_clients (address);
    create index cafe_client_lastSeen on cafe_clients (lastSeen);

    create table cafe_client_threads (id text not null, clientId text not null, ciphertext blob not null, primary key (id, clientId));
    create index cafe_client_thread_clientId on cafe_client_threads (clientId);

    create table cafe_client_messages (id text not null, peerId text not null, clientId text not null, date integer not null, primary key (id, clientId));
    create index cafe_client_message_clientId on cafe_client_messages (clientId);
    create index cafe_client_message_date on cafe_client_messages (date);

    create table cafe_tokens (id text primary key not null, token text not null, date integer not null);
    `
	if _, err := db.Exec(sqlStmt); err != nil {
		return err
	}
	return nil
}
