import { Schema, model } from 'mongoose'

const UserSchema = new Schema(
  {
    role: {
      type: String,
      default: 'user',
      enum: ['user', 'admin'],
    },

    username: { type: String, required: [true, 'Please provide a username'], trim: true },

    email: {
      type: String,
      required: [true, 'Please provide an email'],
      unique: true,
      match: [
        /^(([^<>()\[\]\\.,;:\s@"]+(\.[^<>()\[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/,
        'Please provide a valid email',
      ],
    },

    verified: { type: Boolean, default: false },
    two_fa_status: { type: String, default: 'off' },

    OTP_code: { type: String },

    password: {
      type: String,
      required: [true, 'Please add a password'],
      minlength: 6,
      select: false,
    },
    resetPasswordToken: String,
    resetPasswordExpire: Date,
  },
  { timestamps: true }
)
