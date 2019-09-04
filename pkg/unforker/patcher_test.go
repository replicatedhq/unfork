package unforker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	upstreamFilesFixture = map[string][]byte{
		"deployment.yaml": []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: nginx:1.7.9
          ports:
           - containerPort: 80
`),

		"database.yaml": []byte(`apiVersion: databases.schemahero.io/v1alpha2
kind: Database
metadata:
  name: rds-postgres
  namespace: default
connection:
  postgres:
    uri:
      valueFrom:
        secretKeyRef:
          key: uri
          name: rds-postgres
`),
	}
)

func Test_findMatchingUpstreamPath(t *testing.T) {
	tests := []struct {
		name          string
		upstreamFiles map[string][]byte
		forkedContent []byte
		expected      string
	}{
		{
			name:          "find a deployment",
			upstreamFiles: upstreamFilesFixture,
			forkedContent: []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: default`),
			expected: "deployment.yaml",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			actual, err := findMatchingUpstreamPath(test.upstreamFiles, test.forkedContent)
			req.NoError(err)
			assert.Equal(t, test.expected, actual)
		})
	}
}
