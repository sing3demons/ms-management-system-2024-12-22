import { customAlphabet } from 'nanoid'

const alphanum = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz'

const genNanoId = (size: number) => {
  if (size < 1) {
    return ''
  }
  const nanoid = customAlphabet(alphanum, size)
  return nanoid()
}

export default genNanoId
