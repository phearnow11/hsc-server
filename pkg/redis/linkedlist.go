package redis

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Node struct {
	ID         string
	Data       interface{}
	Next       string
	Expiration time.Duration
}

var HeadNode Node = Node{}

func AddNode(ctx context.Context, prevID string, nodeID string, data interface{}, expiration time.Duration) (string, error) {
	// nodeID = tsfrom-tsto
	if len(strings.Split(nodeID, "-")) != 2 {
		return "", fmt.Errorf("Node key is wrongly set, return without set")
	}

	nodeKey := fmt.Sprint("node:%s", nodeID)

	node := Node{
		ID:         nodeID,
		Data:       data,
		Next:       "",
		Expiration: expiration,
	}

	_, err := client.HSet(ctx, nodeKey, map[string]interface{}{
		"data": node.Data,
		"next": node.Next,
	}).Result()
	if err != nil {
		return "", err
	}

	err = client.Expire(ctx, nodeKey, expiration).Err()
	if err != nil {
		return "", err
	}

	if prevID != "" {
		prevKey := fmt.Sprint("node:%s", prevID)
		err := client.HSet(ctx, prevKey, "next", nodeID).Err()
		if err != nil {
			return "", err
		}
	}

	if HeadNode.Data == nil {
		HeadNode = node
	}

	return nodeID, nil
}

func FindANode(ctx context.Context, from string) (Node, error) {
	curNode := HeadNode
	for {
		if curNode.Next == "" {
			break
		}
	}
	return Node{}, nil
}

func TraverseListRange(ctx context.Context, from string, to string) ([]Node, error) {
	return []Node{}, nil
}
