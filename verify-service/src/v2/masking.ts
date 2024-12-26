import * as crypto from 'crypto'

enum MaskingType {
  password = 'password',
  custom = 'custom',
  msisdn = 'msisdn',
  creditCard = 'creditCard',
  bankAccount = 'bankAccount',
  email = 'email',
  idCard = 'idCard',
  full = 'full',
  firstname = 'firstname',
  lastname = 'lastname',
  hashing = 'hashing',
}

type MaskingRule = {
  maskingField: string
  maskingType: MaskingType
  customMask?: string
}

const HMAC_KEY = process.env.HMAC_KEY_ENV ?? 'default'

class MaskingService {
  constructor(private readonly maskingDisplayCharacter: string = '*') {}

  private censorEmail(email: string): string {
    const [str, domain] = email.split('@')
    if (!str || !domain) return email

    const firstThreeLetters = str.substring(0, 3)
    // Replace all alphanumeric characters, except the first three letters, with 'x'
    const maskedString = str.replace(/[a-zA-Z0-9]/g, (match, index) => {
      return index < 3 ? match : this.maskingDisplayCharacter
    })
    // Combine the first three letters with the masked string
    return firstThreeLetters + maskedString.substring(3) + '@' + domain
  }

  private censorBankAccountNumber(accountNumber: string): string {
    const length = accountNumber.length
    // Extract the first 4 digits and the last 3 digits
    const firstFour = accountNumber.slice(0, 4)
    const lastThree = accountNumber.slice(-3)
    // Replace the remaining digits with "x" characters
    const censoredDigits = this.maskingDisplayCharacter.repeat(length - 7) // Subtracting 7 to account for the 4 visible and 3 visible digits
    // Combine the extracted digits and "x" characters to form the censored bank account number
    const censoredNumber = firstFour + censoredDigits + lastThree
    return censoredNumber
  }

  private censorCreditCardId(cardNumber: string): string {
    if (cardNumber.length < 11) {
      return cardNumber // Return original string if it's too short to censor
    }
    const firstSix = cardNumber.substring(0, 6)
    const lastFour = cardNumber.slice(-4)
    const censored = firstSix + this.maskingDisplayCharacter.repeat(cardNumber.length - 10) + lastFour
    return censored
  }

  private censorPhoneNumber(phoneNumber: string): string {
    if (phoneNumber.length < 7) {
      return phoneNumber // Return original string if it's too short to censor
    }
    const firstThree = phoneNumber.substring(0, 3)
    const lastFour = phoneNumber.slice(-4)
    const censored = firstThree + this.maskingDisplayCharacter.repeat(phoneNumber.length - 7) + lastFour
    return censored
  }

  private censorIDCard(id: string): string {
    if (id.length < 4) {
      return id // Return original string if it's too short to censor
    }
    const lastFour = id.slice(-4)
    const censored = this.maskingDisplayCharacter.repeat(id.length - 4) + lastFour
    return censored
  }

  private censorExcludeFirst3(value: string): string {
    if (value.length < 3) return value
    const first3 = value.substring(0, 3)
    const censored = first3 + this.maskingDisplayCharacter.repeat(value.length - 3)
    return censored
  }

  private censorFull(value: string): string {
    return this.maskingDisplayCharacter.repeat(value.length)
  }

  private hmac(value: string): string {
    try {
      const hmac = crypto.createHmac('md5', HMAC_KEY)
      hmac.update(value)
      const hash = hmac.digest('hex')
      return hash
    } catch (error) {
      throw error
    }
  }

  private setNestedValue(obj: Record<string, any>, path: string, value: any): void {
    const keys = path.split('.')
    let current = obj

    for (let i = 0; i < keys.length - 1; i++) {
      const key = keys[i]
      if (key && current[key]) current = current[key] as Record<string, any>
    }

    const k = keys[keys.length - 1]
    if (k) current[k] = value
  }

  getObjectByStringKeys(obj: Record<string, any>, keysString: string): any {
    const keys = keysString.split('.')
    let currentObj = obj
    for (let key of keys) {
      // Check if the current key is an array index
      if (/\[\d+\]$/.test(key)) {
        const [arrayKey, index] = key.split(/[\[\]]/).filter(Boolean)
        if (arrayKey && index) currentObj = currentObj[arrayKey][parseInt(index, 10)]
      } else {
        currentObj = currentObj[key]
      }
      if (currentObj === undefined) {
        return undefined // Return undefined if any key is not found
      }
    }
    // If the final key represents an array, return all elements
    if (Array.isArray(currentObj)) {
      return currentObj
    }
    return [currentObj] // Wrap non-array values in an array
  }
  private setNestedProperty(obj: Record<string, any>, propString: string, type: MaskingType) {
    const propNames = propString.split('.')
    let currentProp = obj
    for (let i = 0; i < propNames.length - 1; i++) {
      let propName = propNames[i]
      if (propName) {
        if (currentProp[propName]) {
          currentProp = currentProp[propName]
        }
      }
    }

    const key = propNames[propNames.length - 1]
    if (key) {
      const old = currentProp[key]

      if (old) currentProp[key] = this.masking(old, type)
    }
  }

  private setNestedArrayProperty(obj: Record<string, any>, path: string, type: MaskingType) {
    const properties = path.split('.')
    let currentObj = obj
    for (let i = 0; i < properties.length - 1; i++) {
      const property = properties[i]
      if (property) {
        if (currentObj.hasOwnProperty(property)) {
          currentObj = currentObj[property]
        }
      }
    }
    const lastProperty = properties[properties.length - 1]
    if (lastProperty) {
      if (Array.isArray(currentObj[lastProperty])) {
        const index = parseInt(lastProperty)
        if (!isNaN(index) && index >= 0 && index < currentObj[lastProperty].length) {
          const old = currentObj[lastProperty][index]
          if (old) currentObj[lastProperty][index] = this.masking(old, type)
        } else {
          throw new Error('Invalid array index')
        }
      } else {
        const old = currentObj[lastProperty]
        if (old) currentObj[lastProperty] = this.masking(old, type)
      }
    }
  }

  censorship(data: any, opt: MaskingRule) {
    if (opt.maskingField.includes('*')) {
      const k = opt.maskingField.split('*')[0]
      if (k) {
        const root = k.slice(0, -1)
        const lookupArr = this.getObjectByStringKeys(data, root)
        const arrLen = lookupArr.length
        for (let index = 0; index < arrLen; index++) {
          this.setNestedArrayProperty(data, opt.maskingField.replace('*', index + ''), opt.maskingType)
        }
      }
    } else {
      this.setNestedProperty(data, opt.maskingField, opt.maskingType)
    }
  }

  public masking(value: string, type: MaskingType): string {
    switch (type) {
      case MaskingType.msisdn:
        return this.censorPhoneNumber(value)
      case MaskingType.creditCard:
        return this.censorCreditCardId(value)
      case MaskingType.bankAccount:
        return this.censorBankAccountNumber(value)
      case MaskingType.email:
        return this.censorEmail(value)
      case MaskingType.idCard:
        return this.censorIDCard(value)
      case MaskingType.full:
      case MaskingType.password:
        // case MaskingType.custom:
        return this.censorFull(value)
      case MaskingType.firstname:
      case MaskingType.lastname:
        return this.censorExcludeFirst3(value)
      case MaskingType.hashing:
        return this.hmac(value)
      default:
        throw new Error(`Unsupported MaskingType: ${type}`)
    }
  }
}

// Example usage:

const sensitiveData = {
  username: 'john_doe',
  password: 'password',
  email: 'john.doe@example.com',
  profile: {
    firstname: 'John',
    lastname: 'Doexx',
  },
}

function masking(data: any, options: MaskingRule[]) {
  let clonedObject = JSON.parse(JSON.stringify(data))
  if (options) {
    for (const opt of options) {
      const maskingService = new MaskingService()
      maskingService.censorship(clonedObject, opt)
    }
  }
  return clonedObject
}

console.log(
  masking(sensitiveData, [
    { maskingField: 'password', maskingType: MaskingType.password },
    { maskingField: 'email', maskingType: MaskingType.email },
    { maskingField: 'profile.*.firstname', maskingType: MaskingType.firstname },
    { maskingField: 'profile.lastname', maskingType: MaskingType.lastname },
  ])
)

// console.log(maskedData)
// Output:
// {
//   username: 'john_doe',
//   password: '********',
//   email: 'joh*******@example.com',
//   profile: {
//     phone: '081234XXXX',
//     creditCard: '123456********3456',
//   }
// }
