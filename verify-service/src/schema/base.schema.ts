import { Static, Type } from '@sinclair/typebox'

const headerSchema = Type.Object({
  Authorization: Type.String({ pattern: '^Bearer .+$' }),
})

type HeaderType = typeof headerSchema
type HeaderSchemaType = Static<typeof headerSchema>

export { headerSchema, HeaderType, HeaderSchemaType }
