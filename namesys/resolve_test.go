package namesys

import (
	"context"
	"errors"
	"testing"
	"time"

	path "github.com/ipfs/go-ipfs/path"

	testutil "gx/ipfs/QmPdxCaVp4jZ9RbxqZADvKH6kiCR5jHvdR5f2ycjAY6T2a/go-testutil"
	ipns "gx/ipfs/QmRAPFFaF7nrezCZQaLihyp2qbAXqSJU5WpvSpwroMv1Xt/go-ipns"
	peer "gx/ipfs/QmVf8hTAsLLFtn4WPCRNdnaF2Eag2qTBS6uR8AiHPZARXy/go-libp2p-peer"
	mockrouting "gx/ipfs/Qmb1N7zdjG2FexpzWNj8T289u9QnQLEiSsTRadDGQxX32D/go-ipfs-routing/mock"
	ds "gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore"
	dssync "gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore/sync"
)

func TestRoutingResolve(t *testing.T) {
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	serv := mockrouting.NewServer()
	id := testutil.RandIdentityOrFatal(t)
	d := serv.ClientWithDatastore(context.Background(), id, dstore)

	resolver := NewIpnsResolver(d)
	publisher := NewIpnsPublisher(d, dstore)

	privk, pubk, err := testutil.RandTestKeyPair(512)
	if err != nil {
		t.Fatal(err)
	}

	h := path.FromString("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")
	err = publisher.Publish(context.Background(), privk, h)
	if err != nil {
		t.Fatal(err)
	}

	pid, err := peer.IDFromPublicKey(pubk)
	if err != nil {
		t.Fatal(err)
	}

	res, err := resolver.Resolve(context.Background(), pid.Pretty())
	if err != nil {
		t.Fatal(err)
	}

	if res != h {
		t.Fatal("Got back incorrect value.")
	}
}

func TestPrexistingExpiredRecord(t *testing.T) {
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	d := mockrouting.NewServer().ClientWithDatastore(context.Background(), testutil.RandIdentityOrFatal(t), dstore)

	resolver := NewIpnsResolver(d)
	publisher := NewIpnsPublisher(d, dstore)

	privk, pubk, err := testutil.RandTestKeyPair(512)
	if err != nil {
		t.Fatal(err)
	}

	id, err := peer.IDFromPublicKey(pubk)
	if err != nil {
		t.Fatal(err)
	}

	// Make an expired record and put it in the datastore
	h := path.FromString("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")
	eol := time.Now().Add(time.Hour * -1)

	entry, err := ipns.Create(privk, []byte(h), 0, eol)
	if err != nil {
		t.Fatal(err)
	}
	err = PutRecordToRouting(context.Background(), d, pubk, entry)
	if err != nil {
		t.Fatal(err)
	}

	// Now, with an old record in the system already, try and publish a new one
	err = publisher.Publish(context.Background(), privk, h)
	if err != nil {
		t.Fatal(err)
	}

	err = verifyCanResolve(resolver, id.Pretty(), h)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPrexistingRecord(t *testing.T) {
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	d := mockrouting.NewServer().ClientWithDatastore(context.Background(), testutil.RandIdentityOrFatal(t), dstore)

	resolver := NewIpnsResolver(d)
	publisher := NewIpnsPublisher(d, dstore)

	privk, pubk, err := testutil.RandTestKeyPair(512)
	if err != nil {
		t.Fatal(err)
	}

	id, err := peer.IDFromPublicKey(pubk)
	if err != nil {
		t.Fatal(err)
	}

	// Make a good record and put it in the datastore
	h := path.FromString("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")
	eol := time.Now().Add(time.Hour)
	entry, err := ipns.Create(privk, []byte(h), 0, eol)
	if err != nil {
		t.Fatal(err)
	}
	err = PutRecordToRouting(context.Background(), d, pubk, entry)
	if err != nil {
		t.Fatal(err)
	}

	// Now, with an old record in the system already, try and publish a new one
	err = publisher.Publish(context.Background(), privk, h)
	if err != nil {
		t.Fatal(err)
	}

	err = verifyCanResolve(resolver, id.Pretty(), h)
	if err != nil {
		t.Fatal(err)
	}
}

func verifyCanResolve(r Resolver, name string, exp path.Path) error {
	res, err := r.Resolve(context.Background(), name)
	if err != nil {
		return err
	}

	if res != exp {
		return errors.New("got back wrong record")
	}

	return nil
}
