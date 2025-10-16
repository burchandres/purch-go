# Purch

This is a golang version of the Purch backend with improvements to database schema. The [UI](https://github.com/burchandres/ui-purch) (written by the one and only [Sereno Dominguez](https://github.com/seredomi)) is being redone a bit to work with this backend rather than the previous python one. All work will be done in go from here on out.

# Development Setup

This uses [task](https://taskfile.dev/) in place of makefiles. 

If you are a developer ask me for the plaid sandbox API keys and secret. Then you can spin up the docker containers with:

```bash
task build
```
followed by
```bash
task up
```

and that should spin up the `service-purch` and `postgres` containers.

# Feature list

This is meant to serve as a budgeting application first that integrates with [Plaid](https://plaid.com/docs/), providing balances for connected bank accounts and aggregating transactions according to user defined categories.

1. All written in [go](https://go.dev/) with gin as the API framework for performance and maintainability.
2. Persists a user's transactions provided by plaid to power custom made analytics to help empower a user's financial decision making.
3. Allows users to define custom categories where we use semantic search* to align a transaction's plaid provided cateogry label with user defined categories to customize this for the user as much as possible.
4. Allows a user to help mark transactions that were split and what portion of that transaction they're truly responsible for so we can gather accurate trends and provide more tailored recommendations.*
5. 

*= Not yet implemented but on the roadmap for development :)