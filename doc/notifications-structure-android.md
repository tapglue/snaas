# Android notification structure

The current structure of the notification payload on android does not allow for customisation on the client side. When a `notification` object is present the only customisation possible is the notification icon and the primary color.

## Proposed solution

Replace the notification message with a data message. Documentation on both types can be found [here](https://firebase.google.com/docs/cloud-messaging/concept-options#notifications_and_data_messages)

1. Add the data from the `notification` object to the `data` object.
2. Put the `data` object on the top level
2. Remove the `notification` object.

The names of the fields in the `notification` object could be the same in the `data` object. This way the client has to manually handle their notifications, which gives it the liberty to customise the design.

This solution would also be more extensible than the previous solution as it allows for custom fields (i.e. user avatars etc).

