/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package main implements a client for Greeter service.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"

	"github.com/getAlby/lndhub.go/lndhubrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	addr = flag.String("addr", "localhost:10009", "the address to connect to")
)

func main() {
	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := lndhubrpc.NewInvoiceSubscriptionClient(conn)
	id := uint32(115)
	r, err := c.SubsribeInvoices(context.Background(), &lndhubrpc.SubsribeInvoicesRequest{
		FromId: &id,
	})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	fmt.Println("starting loop")
	for {
		result, err := r.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println(err.Error())
		}
		fmt.Println(result)
	}
}
