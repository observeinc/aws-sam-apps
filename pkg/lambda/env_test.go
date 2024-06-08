package lambda_test

import (
	"context"
	"os"
	"testing"

	"github.com/observeinc/aws-sam-apps/pkg/lambda"
)

func TestEnv(t *testing.T) {
	env := struct {
		Embedded struct {
			Test         string `env:"TEST,default=hello"`
			ListOfValues string `env:"LISTOFVALUES"`
		} `env:", prefix=T_"`
	}{}

	if v := os.Getenv("T_TEST"); v != "" {
		t.Fatal("found unexpected environment variable")
	}

	if err := lambda.ProcessEnv(context.Background(), &env); err != nil {
		t.Fatal(err)
	}

	if v := os.Getenv("T_TEST"); v != "hello" {
		t.Fatal("default was not exported back to environment")
	}
}
