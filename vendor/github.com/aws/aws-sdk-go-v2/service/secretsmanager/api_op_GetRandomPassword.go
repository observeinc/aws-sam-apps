// Code generated by smithy-go-codegen DO NOT EDIT.

package secretsmanager

import (
	"context"
	"fmt"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Generates a random password. We recommend that you specify the maximum length
// and include every character type that the system you are generating a password
// for can support. By default, Secrets Manager uses uppercase and lowercase
// letters, numbers, and the following characters in passwords:
// !\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~
//
// Secrets Manager generates a CloudTrail log entry when you call this action.
//
// Required permissions: secretsmanager:GetRandomPassword . For more information,
// see [IAM policy actions for Secrets Manager]and [Authentication and access control in Secrets Manager].
//
// [Authentication and access control in Secrets Manager]: https://docs.aws.amazon.com/secretsmanager/latest/userguide/auth-and-access.html
// [IAM policy actions for Secrets Manager]: https://docs.aws.amazon.com/secretsmanager/latest/userguide/reference_iam-permissions.html#reference_iam-permissions_actions
func (c *Client) GetRandomPassword(ctx context.Context, params *GetRandomPasswordInput, optFns ...func(*Options)) (*GetRandomPasswordOutput, error) {
	if params == nil {
		params = &GetRandomPasswordInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "GetRandomPassword", params, optFns, c.addOperationGetRandomPasswordMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*GetRandomPasswordOutput)
	out.ResultMetadata = metadata
	return out, nil
}

type GetRandomPasswordInput struct {

	// A string of the characters that you don't want in the password.
	ExcludeCharacters *string

	// Specifies whether to exclude lowercase letters from the password. If you don't
	// include this switch, the password can contain lowercase letters.
	ExcludeLowercase *bool

	// Specifies whether to exclude numbers from the password. If you don't include
	// this switch, the password can contain numbers.
	ExcludeNumbers *bool

	// Specifies whether to exclude the following punctuation characters from the
	// password: ! " # $ % & ' ( ) * + , - . / : ; < = > ? @ [ \ ] ^ _ ` { | } ~ . If
	// you don't include this switch, the password can contain punctuation.
	ExcludePunctuation *bool

	// Specifies whether to exclude uppercase letters from the password. If you don't
	// include this switch, the password can contain uppercase letters.
	ExcludeUppercase *bool

	// Specifies whether to include the space character. If you include this switch,
	// the password can contain space characters.
	IncludeSpace *bool

	// The length of the password. If you don't include this parameter, the default
	// length is 32 characters.
	PasswordLength *int64

	// Specifies whether to include at least one upper and lowercase letter, one
	// number, and one punctuation. If you don't include this switch, the password
	// contains at least one of every character type.
	RequireEachIncludedType *bool

	noSmithyDocumentSerde
}

type GetRandomPasswordOutput struct {

	// A string with the password.
	RandomPassword *string

	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationGetRandomPasswordMiddlewares(stack *middleware.Stack, options Options) (err error) {
	if err := stack.Serialize.Add(&setOperationInputMiddleware{}, middleware.After); err != nil {
		return err
	}
	err = stack.Serialize.Add(&awsAwsjson11_serializeOpGetRandomPassword{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsAwsjson11_deserializeOpGetRandomPassword{}, middleware.After)
	if err != nil {
		return err
	}
	if err := addProtocolFinalizerMiddlewares(stack, options, "GetRandomPassword"); err != nil {
		return fmt.Errorf("add protocol finalizers: %v", err)
	}

	if err = addlegacyEndpointContextSetter(stack, options); err != nil {
		return err
	}
	if err = addSetLoggerMiddleware(stack, options); err != nil {
		return err
	}
	if err = addClientRequestID(stack); err != nil {
		return err
	}
	if err = addComputeContentLength(stack); err != nil {
		return err
	}
	if err = addResolveEndpointMiddleware(stack, options); err != nil {
		return err
	}
	if err = addComputePayloadSHA256(stack); err != nil {
		return err
	}
	if err = addRetry(stack, options); err != nil {
		return err
	}
	if err = addRawResponseToMetadata(stack); err != nil {
		return err
	}
	if err = addRecordResponseTiming(stack); err != nil {
		return err
	}
	if err = addSpanRetryLoop(stack, options); err != nil {
		return err
	}
	if err = addClientUserAgent(stack, options); err != nil {
		return err
	}
	if err = smithyhttp.AddErrorCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = addSetLegacyContextSigningOptionsMiddleware(stack); err != nil {
		return err
	}
	if err = addTimeOffsetBuild(stack, c); err != nil {
		return err
	}
	if err = addUserAgentRetryMode(stack, options); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opGetRandomPassword(options.Region), middleware.Before); err != nil {
		return err
	}
	if err = addRecursionDetection(stack); err != nil {
		return err
	}
	if err = addRequestIDRetrieverMiddleware(stack); err != nil {
		return err
	}
	if err = addResponseErrorMiddleware(stack); err != nil {
		return err
	}
	if err = addRequestResponseLogging(stack, options); err != nil {
		return err
	}
	if err = addDisableHTTPSMiddleware(stack, options); err != nil {
		return err
	}
	if err = addSpanInitializeStart(stack); err != nil {
		return err
	}
	if err = addSpanInitializeEnd(stack); err != nil {
		return err
	}
	if err = addSpanBuildRequestStart(stack); err != nil {
		return err
	}
	if err = addSpanBuildRequestEnd(stack); err != nil {
		return err
	}
	return nil
}

func newServiceMetadataMiddleware_opGetRandomPassword(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		OperationName: "GetRandomPassword",
	}
}