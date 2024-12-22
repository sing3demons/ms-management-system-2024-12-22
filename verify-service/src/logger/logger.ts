import { createStream, RotatingFileStream } from 'rotating-file-stream'
import * as os from 'os'
import fs from 'fs'
import { ConfigLog, LogConfig } from './types.js'
const { default: packageJson } = await import('../../package.json', {
  assert: { type: 'json' },
})

const endOfLine = os.EOL
const streamTask = new Map<string, RotatingFileStream>()

const confLog: LogConfig = {
  projectName: packageJson.name,
  namespace: 'default',
  detail: {
    rawData: true,
    path: './logs/detail/',
    console: true,
    file: false,
  },
  summary: {
    path: './logs/summary/',
    console: true,
    file: false,
  },
}

function loadLogConfig(
  config: LogConfig = {
    namespace: 'default',
    projectName: packageJson.name,
    detail: {
      console: true,
      file: false,
      path: './logs/detail/',
    },
    summary: {
      console: true,
      file: false,
      path: './logs/summary/',
    },
  }
): void {
  if (config) {
    if (config.detail) {
      confLog.detail = config.detail
    }
    if (config.summary) {
      confLog.summary = config.summary
    }
  }

  if (confLog.detail.file) {
    if (!fs.existsSync(confLog.detail.path)) {
      fs.mkdirSync(confLog.detail.path, { recursive: true })
    }

    streamTask.set('dtl', createStreams('dtl'))
  }

  if (confLog.summary.file) {
    if (!fs.existsSync(confLog.summary.path)) {
      fs.mkdirSync(confLog.summary.path, { recursive: true })
    }

    streamTask.set('smr', createStreams('smr'))
  }
}

function writeLogFile(type: 'smr' | 'dtl', log: string) {
  if (streamTask.get(type)) {
    streamTask.get(type)?.write(log + endOfLine)
  }
}

if (process.env.CONFIG_LOG) {
  const configLog = JSON.parse(process.env.CONFIG_LOG)

  const updateConfig = (target: ConfigLog, source: Partial<ConfigLog>) => {
    Object.assign(target, source)
  }

  if (configLog.projectName) {
    confLog.projectName = configLog.projectName
  }
  if (configLog.namespace) {
    confLog.namespace = configLog.namespace
  }
  if (configLog.summary) {
    updateConfig(confLog.summary, configLog.summary)
  }
  if (configLog.detail) {
    updateConfig(confLog.detail, configLog.detail)
  }
}

function getFileName(type: 'smr' | 'dtl', date?: Date | undefined, index?: number | undefined): string {
  const hostname = os.hostname()
  const projectName = confLog.projectName
  const pmId = process.pid
  if (!date) {
    date = new Date()
  }

  const formattedDate = () => {
    const year = date.getFullYear()
    const month = `0${date.getMonth() + 1}`.slice(-2)
    const day = `0${date.getDate()}`.slice(-2)
    const hour = `0${date.getHours()}`.slice(-2)
    const minute = `0${date.getMinutes()}`.slice(-2)
    const second = `0${date.getSeconds()}`.slice(-2)
    return `${year}${month}${day}${hour}${minute}${second}`
  }
  const formattedIndex = index ? `.${index}` : ''
  if (type === 'smr') {
    return `/${hostname}_${projectName}_${formattedDate()}${formattedIndex}.${pmId}.sum.log`
  }

  return `/${hostname}_${projectName}_${formattedDate()}${formattedIndex}.${pmId}.detail.log`
}

function createStreams(type: 'smr' | 'dtl') {
  const stream = createStream(getFileName(type), {
    size: '10M', // rotate every 10 MegaBytes written
    interval: '1d', // rotate daily
    compress: 'gzip', // compress rotated files
    path: type === 'smr' ? confLog.summary.path : confLog.detail.path,
  })

  stream.on('error', (err) => {
    console.error(err)
  })

  stream.on('warning', (err) => {
    console.error(err)
  })

  return stream
}

export { getFileName, createStreams, confLog, loadLogConfig, writeLogFile }
