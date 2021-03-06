package gossip

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/republicprotocol/babble-go/core/addr"
	"github.com/republicprotocol/co-go"
)

// An Observer is notified whenever a new Message, or an update to an existing
// Message, is received.
type Observer interface {
	Notify(message Message) error
}

// A Signer can consume bytes and produce a signature for those bytes. This
// signature can be used by a Verifier to extract the signatory.
type Signer interface {
	Sign(data []byte) ([]byte, error)
}

// A Verifier can consume bytes and a signature for those bytes, and extract
// the signatory.
type Verifier interface {
	Verify(data []byte, signature []byte) error
}

// A Client is used to send Store to a remote Server.
type Client interface {

	// Send a Message to the a remote `net.Addr`.
	Send(ctx context.Context, to net.Addr, message Message) error
}

// A Server receives Store.
type Server interface {

	// Receive is called to notify the Server that a Message has been received
	// from a remote Client.
	Receive(ctx context.Context, message Message) error
}

// Gossiper is a participant in the gossip network. It can receive message and
// broadcast new message to the network.
type Gossiper interface {
	Server
	Broadcast(ctx context.Context, message Message) error
}

type gossiper struct {
	addrBook addr.Book
	α        int

	signer   Signer
	verifier Verifier
	observer Observer
	client   Client
	messages Messages
}

// NewGossiper returns a new gosspier.
func NewGossiper(addrBook addr.Book, α int, signer Signer, verifier Verifier, observer Observer, client Client, messages Messages) Gossiper {
	return &gossiper{
		addrBook: addrBook,
		α:        α,

		signer:   signer,
		verifier: verifier,
		observer: observer,
		client:   client,
		messages: messages,
	}
}

// Broadcast implements the Gossiper interface.
func (gossiper *gossiper) Broadcast(ctx context.Context, message Message) error {
	return gossiper.broadcast(ctx, message, true)
}

// Receive implements the Gossiper interface.
func (gossiper *gossiper) Receive(ctx context.Context, message Message) error {
	if err := gossiper.verifier.Verify(message.Value, message.Signature); err != nil {
		return err
	}

	previousMessage, err := gossiper.messages.Message(message.Key)
	if err != nil {
		return err
	}
	if previousMessage.Nonce >= message.Nonce {
		return nil
	}
	if err := gossiper.messages.InsertMessage(message); err != nil {
		return err
	}

	if gossiper.observer != nil {
		if err := gossiper.observer.Notify(message); err != nil {
			return err
		}
	}

	return gossiper.broadcast(ctx, message, false)
}

func (gossiper *gossiper) broadcast(ctx context.Context, message Message, sign bool) error {
	if sign {
		signature, err := gossiper.signer.Sign(message.Value)
		if err != nil {
			return err
		}
		message.Signature = signature
	}

	addrs, err := gossiper.addrBook.Addrs(gossiper.α)
	if err != nil {
		return err
	}

	go co.ForAll(addrs, func(i int) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		err := gossiper.client.Send(ctx, addrs[i], message)
		if err != nil {
			// TODO : config the logger
			log.Printf("[error] cannot send messge to %v = %v", addrs[i].String(), err)
		}
	})

	return nil
}
