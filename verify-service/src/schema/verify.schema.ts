import { Static, Type } from '@sinclair/typebox'

const verifySchema = Type.Object({
  email: Type.Optional(Type.String()),
  username: Type.Optional(Type.String()),
  id: Type.String(),
  x :Type.String()
})

type VerifyType = typeof verifySchema
type VerifySchemaType = Static<typeof verifySchema>

export { verifySchema, VerifyType, VerifySchemaType }
