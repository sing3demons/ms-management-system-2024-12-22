import randomString from 'randomstring'
import dayjs from 'dayjs'
import type { LogConfig } from '../config.js'
const { default: packageJson } = await import('../../../package.json', {
  assert: { type: 'json' },
})


function generateXTid(nodeName: string = packageJson.name): string {
  const now = new Date()
  const date = dayjs(now, 'yymmdd')
  let xTid = nodeName.substring(0, 5) + '-' + date
  const remainingLength = 22 - xTid.length
  xTid += randomString.generate(remainingLength)
  return xTid
}


const projectName = packageJson.name

export { randomString, generateXTid, LogConfig, projectName }
