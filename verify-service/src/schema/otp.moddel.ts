import { Schema, model } from 'mongoose'

const otpSchema = new Schema({
  userId: {
    type: String,
    required: true,
    index: true,
    unique: true,
    sparse: true,
  },
  otp: { type: String, required: true },
  createdAt: { type: Date, default: Date.now, expires: 3600 },
})

const otpModel = model('otp', otpSchema)

export default otpModel
