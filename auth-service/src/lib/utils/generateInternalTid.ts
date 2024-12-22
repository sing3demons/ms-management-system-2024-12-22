import moment from 'moment'
import genNanoId from './nanoid.js'

function generateInternalTid(node_name: string, symbol: string, length: number) {
  let internalTid = ''
  let lengthOfNodeName = node_name.length
  if (lengthOfNodeName > 5) {
    node_name = node_name.substring(0, 5)
    lengthOfNodeName = 5
  }

  if (length > 0) {
    const lengthOfSymbol = symbol.length
    const currentDate = moment().format('YYMMDD')
    const digitsNonDate = length - currentDate.length
    if (digitsNonDate > 0) {
      const randomDigit = digitsNonDate - (lengthOfNodeName + lengthOfSymbol)
      if (randomDigit > 0) {
        internalTid = node_name + symbol + currentDate + genNanoId(randomDigit)
      } else {
        internalTid = node_name + symbol + currentDate
      }
    }
  } else {
    internalTid = genNanoId(length)
  }
  return internalTid
}

export default generateInternalTid
