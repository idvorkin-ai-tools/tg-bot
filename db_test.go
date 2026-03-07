package main

import (
	"os"
	"testing"
)

func testDB(t *testing.T) *DB {
	t.Helper()
	path := t.TempDir() + "/test.db"
	db, err := OpenDB(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		os.Remove(path)
	})
	return db
}

func TestInsertAndPollMessages(t *testing.T) {
	db := testDB(t)

	err := db.InsertMessage(Message{
		TelegramMsgID: 1,
		ChatID:        100,
		TopicID:       nil,
		SenderName:    "Alice",
		SenderID:      42,
		Content:       "hello",
		Timestamp:     "2026-03-07T12:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}

	msgs, err := db.PollMessages(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Content != "hello" {
		t.Fatalf("expected 'hello', got %q", msgs[0].Content)
	}
	if msgs[0].SenderName != "Alice" {
		t.Fatalf("expected 'Alice', got %q", msgs[0].SenderName)
	}

	// polling again should return nothing (marked as read)
	msgs2, err := db.PollMessages(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs2) != 0 {
		t.Fatalf("expected 0 messages after poll, got %d", len(msgs2))
	}
}

func TestPollMessagesByChatID(t *testing.T) {
	db := testDB(t)

	db.InsertMessage(Message{TelegramMsgID: 1, ChatID: 100, SenderName: "A", SenderID: 1, Content: "msg1", Timestamp: "2026-03-07T12:00:00Z"})
	db.InsertMessage(Message{TelegramMsgID: 2, ChatID: 200, SenderName: "B", SenderID: 2, Content: "msg2", Timestamp: "2026-03-07T12:01:00Z"})

	msgs, err := db.PollMessages(100)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message for chat 100, got %d", len(msgs))
	}
	if msgs[0].Content != "msg1" {
		t.Fatalf("expected 'msg1', got %q", msgs[0].Content)
	}
}

func TestUpsertAndListChats(t *testing.T) {
	db := testDB(t)

	err := db.UpsertChat(Chat{
		ChatID:   100,
		ChatType: "group",
		Title:    "Test Group",
		IsForum:  true,
		LastSeen: "2026-03-07T12:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}

	chats, err := db.ListChats()
	if err != nil {
		t.Fatal(err)
	}
	if len(chats) != 1 {
		t.Fatalf("expected 1 chat, got %d", len(chats))
	}
	if chats[0].Title != "Test Group" {
		t.Fatalf("expected 'Test Group', got %q", chats[0].Title)
	}
	if !chats[0].IsForum {
		t.Fatal("expected IsForum=true")
	}
}
