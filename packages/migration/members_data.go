package migration

import "github.com/GenesisKernel/go-genesis/packages/consts"

var membersDataSQL = `
	INSERT INTO "1_members" ("id", "member_name", "ecosystem") VALUES('%[2]d', 'founder', '%[1]d'),
	('` + consts.GuestKey + `', 'guest', '%[1]d');

`