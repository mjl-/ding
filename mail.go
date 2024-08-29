package main

import (
	"fmt"
)

func repoRecipients(r Repo) []mailAddr {
	if len(r.NotifyEmailAddrs) == 0 {
		return []mailAddr{{Name: config.Notify.Name, Address: config.Notify.Email}}
	}
	rcpts := make([]mailAddr, len(r.NotifyEmailAddrs))
	for i, addr := range r.NotifyEmailAddrs {
		rcpts[i] = mailAddr{Address: addr}
	}
	return rcpts
}

func _sendMailFailing(repo Repo, build Build, errmsg string) {
	link := fmt.Sprintf("%s/#/repo/%s/build/%d/", config.BaseURL, repo.Name, build.ID)
	subject := fmt.Sprintf("ding: failure: repo %s branch %s failing", repo.Name, build.Branch)
	textMsg := fmt.Sprintf(`Hi!

Your build for branch %s on repo %s is now failing:

	%s

Last output:

	%s
	%s

Please fix, thanks!

Cheers,
Ding
`, build.Branch, repo.Name, link, build.LastLine, errmsg)

	_sendmail(repoRecipients(repo), subject, textMsg)
}

func _sendMailFixed(repo Repo, build Build) {
	link := fmt.Sprintf("%s/#/repo/%s/build/%d/", config.BaseURL, repo.Name, build.ID)
	subject := fmt.Sprintf("ding: resolved: repo %s branch %s is building again", repo.Name, build.Branch)
	textMsg := fmt.Sprintf(`Hi!

You fixed the build for branch %s on repo %s:

	%s

You're the bomb, keep it up!

Cheers,
Ding
`, build.Branch, repo.Name, link)

	_sendmail(repoRecipients(repo), subject, textMsg)
}
