package main

import (
	"testing"
)

func TestMail(t *testing.T) {
	testEnv(t)

	client := &fakeClient{true, nil}
	newSMTPClient = func() smtpClient { return client }

	api := Ding{}

	// Check email is sent when build starts failing.
	r := Repo{Name: "mailtest", VCS: VCSCommand, Origin: "exit 1", DefaultBranch: "main", CheckoutPath: "mailtest", BuildScript: "#!/bin/bash\necho hi\n"}
	r = api.RepoCreate(ctxbg, config.Password, r)
	b := api.BuildCreate(ctxbg, config.Password, r.Name, "unused", "", false)
	twaitBuild(t, b, StatusClone)
	tcompare(t, client.recipients, []string{config.Notify.Email})
	client.recipients = nil

	// And see we get a mail when fixed again.
	r.Origin = "echo clone..; mkdir -p checkout/$DING_CHECKOUTPATH; echo commit: ..."
	r.NotifyEmailAddrs = []string{"addr1@ding.example", "addr2@ding.example"}
	r = api.RepoSave(ctxbg, config.Password, r)
	b = api.BuildCreate(ctxbg, config.Password, r.Name, "unused", "", false)
	twaitBuild(t, b, StatusSuccess)
	tcompare(t, client.recipients, r.NotifyEmailAddrs)
	client.recipients = nil
}
