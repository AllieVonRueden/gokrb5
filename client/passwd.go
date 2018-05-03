package client

import (
	"fmt"
	"net"

	"gopkg.in/jcmturner/gokrb5.v4/kadmin"
)

const (
	passwdChangeSPN = "kadmin/changepw"

	KRB5_KPASSWD_SUCCESS             = 0
	KRB5_KPASSWD_MALFORMED           = 1
	KRB5_KPASSWD_HARDERROR           = 2
	KRB5_KPASSWD_AUTHERROR           = 3
	KRB5_KPASSWD_SOFTERROR           = 4
	KRB5_KPASSWD_ACCESSDENIED        = 5
	KRB5_KPASSWD_BAD_VERSION         = 6
	KRB5_KPASSWD_INITIAL_FLAG_NEEDED = 7
)

func (cl *Client) ChangePasswd(newPasswd string) (bool, error) {
	tkt, skey, err := cl.GetServiceTicket(passwdChangeSPN)
	if err != nil {
		return false, fmt.Errorf("could not get service ticket: %v", err)
	}

	msg, key, err := kadmin.ChangePasswdMsg(cl.Credentials.CName, cl.Credentials.Realm, newPasswd, tkt, skey)
	r, err := cl.sendToKPasswd(msg)
	err = r.Decrypt(key)
	if err != nil {
		return false, err
	}
	if r.ResultCode != KRB5_KPASSWD_SUCCESS {
		return false, fmt.Errorf("error response from kdamin: %s", r.Result)
	}
	return true, nil
}

func (cl *Client) sendToKPasswd(msg kadmin.Request) (r kadmin.Reply, err error) {
	_, kps, err := cl.Config.GetKpasswdServers(cl.Credentials.Realm, true)
	if err != nil {
		return
	}
	addr := kps[1]
	b, err := msg.Marshal()
	if err != nil {
		return
	}
	if len(b) <= cl.Config.LibDefaults.UDPPreferenceLimit {
		return cl.sendKPasswdUDP(b, addr)
	}
	return cl.sendKPasswdTCP(b, addr)
}

func (cl *Client) sendKPasswdTCP(b []byte, kadmindAddr string) (r kadmin.Reply, err error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", kadmindAddr)
	if err != nil {
		return
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return
	}
	rb, err := cl.sendTCP(conn, b)
	err = r.Unmarshal(rb)
	return
}

func (cl *Client) sendKPasswdUDP(b []byte, kadmindAddr string) (r kadmin.Reply, err error) {
	udpAddr, err := net.ResolveUDPAddr("udp", kadmindAddr)
	if err != nil {
		return
	}
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return
	}
	rb, err := cl.sendUDP(conn, b)
	err = r.Unmarshal(rb)
	return
}