# Flexible Messaging

In its first inception SIMS (Signal informed Messaging Service) showed some significant weaknesses. With the biggest one being, due to sending unfiltered notifications for every transaction, a quick developing fatigue for the user up to outright annoyance and/or perceiving the messages as spam. On the other hand is the generic content in the messages themselves decreasing the relevance and usually fall apart quickly in specialised applications.

We've proven that the system is capable of reacting to changes in apps in a responsive manner so that we now need to address the quality of notifications. In order to do this we going to overhaul the pipeline of how changes are consumed in SIMS and introduce a couple of new concepts along the way. The goal is to empower the operator with little effort to create quality notifications.

At the heart of the new pipeline will be rules and templates. Where rules are the configuration based on the input e.g (a new `Post` which has the `article` tag). After creation of such rule it can be associated with a template which can make use of variables provided (e.g. `recipient.Username`).

### Components

Some house-keeping has to be done as we haven't followed through with some of the concepts required. Up until now we hard-coded the mappings of platform information to internal understanding of an App. As we use `SNS` there is quite some management going on and we need to put that information (certs, endpoint, schema) in a persistent place that can be managed without code deploys or issuing `SQL` statements.

#### Pipeline

* **rule**: Determines if and who message should be send to based on configurable criteria and what the message content looks like.
* **criteria**: Old and new entity information which can be used in conditions.
* **recipient**: Users which stand in relation to the entity (e.g. owner of a post).
* **template**: Interpolate string which has a set of variables to work with and is created per recipient and language.
* **var**: Piece of information that can be used in templates for personalisation.


#### Platform

Is the representation of an AWS Platform Application and is used to track important information and alleviate the need for direct interaction with SNS.

``` go
// Platform supported for a Device.
const (
	IOSSandbox Ecosystem = iota + 1
	IOS
	Android
)

// Ecosystem of a device.
type Ecosystem uint8

// Platform represents an ecosystem like Android or iOS.
type Platform struct {
	Active    bool
	ARN       string
	Ecosystem Ecosystem
	Scheme    string
}
```

### Tasks

- [ ] Implement Platform
- [ ] Implement Rules
- [ ] Update SIMS