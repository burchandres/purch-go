# Purch

This is a golang version of the Purch backend with improvements to database schema. The [UI](https://github.com/burchandres/ui-purch) (written by the one and only [Sereno Dominguez](https://github.com/seredomi)) is being redone a bit to work with this backend rather than the previous python one. All work will be done in go from here on out.

# Development Setup

This uses [task](https://taskfile.dev/) in place of makefiles. 

If you are a developer ask me for the plaid sandbox API keys and secret. Paste them into your `.env` file which you get by copying the template:

```bash
cp .env.template .env
```

Then you can spin up the docker containers with:

```bash
task build
```
followed by
```bash
task up
```

and that should spin up the `service-purch`, `service-purch-webhook` and `postgres` containers.

The `service-purch-webhook` binary is compiled from the `webhook/cmd/main.go` file, and the `service-purch` binary is compiled from the root `main.go` file.  The `service-purch-webhook` server exists on a separate port so that when exposed to the internet for plaid webhook integration the main server doesn't get DDOS'd.

# Feature list

This is meant to serve as a budgeting application that integrates with [Plaid](https://plaid.com/docs/), providing balances for connected bank accounts and aggregating transactions according to user defined categories.

1. All written in [go](https://go.dev/) with gin as the API framework for performance and maintainability and postgres as the transactional database for user data persistence.
2. Allows users to define custom categories where we use semantic search* to align a transaction's plaid provided cateogry label with user defined categories to customize this for the user as much as possible.
3. Allows a user to help mark transactions that were split and what portion of that transaction they're truly responsible for so we can gather accurate trends and provide more tailored recommendations.*
4. Supports checking, savings and credit card accounts.

*= Not yet implemented but on the roadmap for development :)
