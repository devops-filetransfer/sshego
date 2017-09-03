// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssh

import (
	"bytes"
	"context"
	"crypto/rand"
	"reflect"
	"testing"
	"time"
)

// Cert generated by ssh-keygen 6.0p1 Debian-4.
// % ssh-keygen -s ca-key -I test user-key
const exampleSSHCert = `ssh-rsa-cert-v01@openssh.com AAAAHHNzaC1yc2EtY2VydC12MDFAb3BlbnNzaC5jb20AAAAgb1srW/W3ZDjYAO45xLYAwzHBDLsJ4Ux6ICFIkTjb1LEAAAADAQABAAAAYQCkoR51poH0wE8w72cqSB8Sszx+vAhzcMdCO0wqHTj7UNENHWEXGrU0E0UQekD7U+yhkhtoyjbPOVIP7hNa6aRk/ezdh/iUnCIt4Jt1v3Z1h1P+hA4QuYFMHNB+rmjPwAcAAAAAAAAAAAAAAAEAAAAEdGVzdAAAAAAAAAAAAAAAAP//////////AAAAAAAAAIIAAAAVcGVybWl0LVgxMS1mb3J3YXJkaW5nAAAAAAAAABdwZXJtaXQtYWdlbnQtZm9yd2FyZGluZwAAAAAAAAAWcGVybWl0LXBvcnQtZm9yd2FyZGluZwAAAAAAAAAKcGVybWl0LXB0eQAAAAAAAAAOcGVybWl0LXVzZXItcmMAAAAAAAAAAAAAAHcAAAAHc3NoLXJzYQAAAAMBAAEAAABhANFS2kaktpSGc+CcmEKPyw9mJC4nZKxHKTgLVZeaGbFZOvJTNzBspQHdy7Q1uKSfktxpgjZnksiu/tFF9ngyY2KFoc+U88ya95IZUycBGCUbBQ8+bhDtw/icdDGQD5WnUwAAAG8AAAAHc3NoLXJzYQAAAGC8Y9Z2LQKhIhxf52773XaWrXdxP0t3GBVo4A10vUWiYoAGepr6rQIoGGXFxT4B9Gp+nEBJjOwKDXPrAevow0T9ca8gZN+0ykbhSrXLE5Ao48rqr3zP4O1/9P7e6gp0gw8=`

func TestParseCert(t *testing.T) {
	defer xtestend(xtestbegin(t))

	authKeyBytes := []byte(exampleSSHCert)

	key, _, _, rest, err := ParseAuthorizedKey(authKeyBytes)
	if err != nil {
		t.Fatalf("ParseAuthorizedKey: %v", err)
	}
	if len(rest) > 0 {
		t.Errorf("rest: got %q, want empty", rest)
	}

	if _, ok := key.(*Certificate); !ok {
		t.Fatalf("got %v (%T), want *Certificate", key, key)
	}

	marshaled := MarshalAuthorizedKey(key)
	// Before comparison, remove the trailing newline that
	// MarshalAuthorizedKey adds.
	marshaled = marshaled[:len(marshaled)-1]
	if !bytes.Equal(authKeyBytes, marshaled) {
		t.Errorf("marshaled certificate does not match original: got %q, want %q", marshaled, authKeyBytes)
	}
}

// Cert generated by ssh-keygen OpenSSH_6.8p1 OS X 10.10.3
// % ssh-keygen -s ca -I testcert -O source-address=192.168.1.0/24 -O force-command=/bin/sleep user.pub
// user.pub key: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDACh1rt2DXfV3hk6fszSQcQ/rueMId0kVD9U7nl8cfEnFxqOCrNT92g4laQIGl2mn8lsGZfTLg8ksHq3gkvgO3oo/0wHy4v32JeBOHTsN5AL4gfHNEhWeWb50ev47hnTsRIt9P4dxogeUo/hTu7j9+s9lLpEQXCvq6xocXQt0j8MV9qZBBXFLXVT3cWIkSqOdwt/5ZBg+1GSrc7WfCXVWgTk4a20uPMuJPxU4RQwZW6X3+O8Pqo8C3cW0OzZRFP6gUYUKUsTI5WntlS+LAxgw1mZNsozFGdbiOPRnEryE3SRldh9vjDR3tin1fGpA5P7+CEB/bqaXtG3V+F2OkqaMN
// Critical Options:
//         force-command /bin/sleep
//         source-address 192.168.1.0/24
// Extensions:
//         permit-X11-forwarding
//         permit-agent-forwarding
//         permit-port-forwarding
//         permit-pty
//         permit-user-rc
const exampleSSHCertWithOptions = `ssh-rsa-cert-v01@openssh.com AAAAHHNzaC1yc2EtY2VydC12MDFAb3BlbnNzaC5jb20AAAAgDyysCJY0XrO1n03EeRRoITnTPdjENFmWDs9X58PP3VUAAAADAQABAAABAQDACh1rt2DXfV3hk6fszSQcQ/rueMId0kVD9U7nl8cfEnFxqOCrNT92g4laQIGl2mn8lsGZfTLg8ksHq3gkvgO3oo/0wHy4v32JeBOHTsN5AL4gfHNEhWeWb50ev47hnTsRIt9P4dxogeUo/hTu7j9+s9lLpEQXCvq6xocXQt0j8MV9qZBBXFLXVT3cWIkSqOdwt/5ZBg+1GSrc7WfCXVWgTk4a20uPMuJPxU4RQwZW6X3+O8Pqo8C3cW0OzZRFP6gUYUKUsTI5WntlS+LAxgw1mZNsozFGdbiOPRnEryE3SRldh9vjDR3tin1fGpA5P7+CEB/bqaXtG3V+F2OkqaMNAAAAAAAAAAAAAAABAAAACHRlc3RjZXJ0AAAAAAAAAAAAAAAA//////////8AAABLAAAADWZvcmNlLWNvbW1hbmQAAAAOAAAACi9iaW4vc2xlZXAAAAAOc291cmNlLWFkZHJlc3MAAAASAAAADjE5Mi4xNjguMS4wLzI0AAAAggAAABVwZXJtaXQtWDExLWZvcndhcmRpbmcAAAAAAAAAF3Blcm1pdC1hZ2VudC1mb3J3YXJkaW5nAAAAAAAAABZwZXJtaXQtcG9ydC1mb3J3YXJkaW5nAAAAAAAAAApwZXJtaXQtcHR5AAAAAAAAAA5wZXJtaXQtdXNlci1yYwAAAAAAAAAAAAABFwAAAAdzc2gtcnNhAAAAAwEAAQAAAQEAwU+c5ui5A8+J/CFpjW8wCa52bEODA808WWQDCSuTG/eMXNf59v9Y8Pk0F1E9dGCosSNyVcB/hacUrc6He+i97+HJCyKavBsE6GDxrjRyxYqAlfcOXi/IVmaUGiO8OQ39d4GHrjToInKvExSUeleQyH4Y4/e27T/pILAqPFL3fyrvMLT5qU9QyIt6zIpa7GBP5+urouNavMprV3zsfIqNBbWypinOQAw823a5wN+zwXnhZrgQiHZ/USG09Y6k98y1dTVz8YHlQVR4D3lpTAsKDKJ5hCH9WU4fdf+lU8OyNGaJ/vz0XNqxcToe1l4numLTnaoSuH89pHryjqurB7lJKwAAAQ8AAAAHc3NoLXJzYQAAAQCaHvUIoPL1zWUHIXLvu96/HU1s/i4CAW2IIEuGgxCUCiFj6vyTyYtgxQxcmbfZf6eaITlS6XJZa7Qq4iaFZh75C1DXTX8labXhRSD4E2t//AIP9MC1rtQC5xo6FmbQ+BoKcDskr+mNACcbRSxs3IL3bwCfWDnIw2WbVox9ZdcthJKk4UoCW4ix4QwdHw7zlddlz++fGEEVhmTbll1SUkycGApPFBsAYRTMupUJcYPIeReBI/m8XfkoMk99bV8ZJQTAd7OekHY2/48Ff53jLmyDjP7kNw1F8OaPtkFs6dGJXta4krmaekPy87j+35In5hFj7yoOqvSbmYUkeX70/GGQ`

func TestParseCertWithOptions(t *testing.T) {
	defer xtestend(xtestbegin(t))

	opts := map[string]string{
		"source-address": "192.168.1.0/24",
		"force-command":  "/bin/sleep",
	}
	exts := map[string]string{
		"permit-X11-forwarding":   "",
		"permit-agent-forwarding": "",
		"permit-port-forwarding":  "",
		"permit-pty":              "",
		"permit-user-rc":          "",
	}
	authKeyBytes := []byte(exampleSSHCertWithOptions)

	key, _, _, rest, err := ParseAuthorizedKey(authKeyBytes)
	if err != nil {
		t.Fatalf("ParseAuthorizedKey: %v", err)
	}
	if len(rest) > 0 {
		t.Errorf("rest: got %q, want empty", rest)
	}
	cert, ok := key.(*Certificate)
	if !ok {
		t.Fatalf("got %v (%T), want *Certificate", key, key)
	}
	if !reflect.DeepEqual(cert.CriticalOptions, opts) {
		t.Errorf("unexpected critical options - got %v, want %v", cert.CriticalOptions, opts)
	}
	if !reflect.DeepEqual(cert.Extensions, exts) {
		t.Errorf("unexpected Extensions - got %v, want %v", cert.Extensions, exts)
	}
	marshaled := MarshalAuthorizedKey(key)
	// Before comparison, remove the trailing newline that
	// MarshalAuthorizedKey adds.
	marshaled = marshaled[:len(marshaled)-1]
	if !bytes.Equal(authKeyBytes, marshaled) {
		t.Errorf("marshaled certificate does not match original: got %q, want %q", marshaled, authKeyBytes)
	}
}

func TestValidateCert(t *testing.T) {
	defer xtestend(xtestbegin(t))

	key, _, _, _, err := ParseAuthorizedKey([]byte(exampleSSHCert))
	if err != nil {
		t.Fatalf("ParseAuthorizedKey: %v", err)
	}
	validCert, ok := key.(*Certificate)
	if !ok {
		t.Fatalf("got %v (%T), want *Certificate", key, key)
	}
	checker := CertChecker{}
	checker.IsUserAuthority = func(k PublicKey) bool {
		return bytes.Equal(k.Marshal(), validCert.SignatureKey.Marshal())
	}

	if err := checker.CheckCert("user", validCert); err != nil {
		t.Errorf("Unable to validate certificate: %v", err)
	}
	invalidCert := &Certificate{
		Key:          testPublicKeys["rsa"],
		SignatureKey: testPublicKeys["ecdsa"],
		ValidBefore:  CertTimeInfinity,
		Signature:    &Signature{},
	}
	if err := checker.CheckCert("user", invalidCert); err == nil {
		t.Error("Invalid cert signature passed validation")
	}
}

func TestValidateCertTime(t *testing.T) {
	defer xtestend(xtestbegin(t))

	cert := Certificate{
		ValidPrincipals: []string{"user"},
		Key:             testPublicKeys["rsa"],
		ValidAfter:      50,
		ValidBefore:     100,
	}

	cert.SignCert(rand.Reader, testSigners["ecdsa"])

	for ts, ok := range map[int64]bool{
		25:  false,
		50:  true,
		99:  true,
		100: false,
		125: false,
	} {
		checker := CertChecker{
			Clock: func() time.Time { return time.Unix(ts, 0) },
		}
		checker.IsUserAuthority = func(k PublicKey) bool {
			return bytes.Equal(k.Marshal(),
				testPublicKeys["ecdsa"].Marshal())
		}

		if v := checker.CheckCert("user", &cert); (v == nil) != ok {
			t.Errorf("Authenticate(%d): %v", ts, v)
		}
	}
}

// TODO(hanwen): tests for
//
// host keys:
// * fallbacks

func TestHostKeyCert(t *testing.T) {
	defer xtestend(xtestbegin(t))

	cert := &Certificate{
		ValidPrincipals: []string{"hostname", "hostname.domain", "otherhost"},
		Key:             testPublicKeys["rsa"],
		ValidBefore:     CertTimeInfinity,
		CertType:        HostCert,
	}
	cert.SignCert(rand.Reader, testSigners["ecdsa"])

	checker := &CertChecker{
		IsHostAuthority: func(p PublicKey, addr string) bool {
			return addr == "hostname:22" && bytes.Equal(testPublicKeys["ecdsa"].Marshal(), p.Marshal())
		},
	}

	halt := NewHalter()
	defer halt.RequestStop()
	certSigner, err := NewCertSigner(cert, testSigners["rsa"])
	if err != nil {
		t.Errorf("NewCertSigner: %v", err)
	}

	for _, test := range []struct {
		addr    string
		succeed bool
	}{
		{addr: "hostname:22", succeed: true},
		{addr: "otherhost:22", succeed: false}, // The certificate is valid for 'otherhost' as hostname, but we only recognize the authority of the signer for the address 'hostname:22'
		{addr: "lasthost:22", succeed: false},
	} {
		c1, c2, err := netPipe()
		if err != nil {
			t.Fatalf("netPipe: %v", err)
		}
		defer c1.Close()
		defer c2.Close()

		errc := make(chan error)
		ctx := context.Background()
		go func() {
			conf := ServerConfig{
				NoClientAuth: true,
				Config: Config{
					Halt: halt,
				},
			}
			conf.AddHostKey(certSigner)
			_, _, _, err := NewServerConn(ctx, c1, &conf)
			errc <- err
		}()

		config := &ClientConfig{
			User:            "user",
			HostKeyCallback: checker.CheckHostKey,
			Config: Config{
				Halt: halt,
			},
		}
		_, _, _, err = NewClientConn(ctx, c2, test.addr, config)
		defer config.Halt.RequestStop()

		if (err == nil) != test.succeed {
			t.Fatalf("NewClientConn(%q): %v", test.addr, err)
		}

		err = <-errc
		if (err == nil) != test.succeed {
			t.Fatalf("NewServerConn(%q): %v", test.addr, err)
		}
	}
}
