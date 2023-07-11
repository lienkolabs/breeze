package main

const helpdoc = `
Auth is a tool for managing crypto keys for social protocols in the breeze ecosystem.

Usage: 

	auth @handle <command> [arguments]

The commands are:

	enter          register @handle on the axe protocol with a new keypair
	update         update info about the @handle
	grant          grant power of attorney rights 
	revoke         revoke power of attorney rights
	show-attorneys show the list of attorneys
	create-stage   create new stage
	update-stage   update stage-info
	request-stage  request participation on a stage
	accept-request accept-request for participation on stage
	rotate-keys    rotate crypto keys of stage

Use "auth help <command>" for more information about a command. 
`
