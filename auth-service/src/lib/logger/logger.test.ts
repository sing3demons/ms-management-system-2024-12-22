import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import * as rfs from 'rotating-file-stream'
import { confLog, writeLogFile, loadLogConfig } from "./logger";
import { IConfig } from "../config";
import fs from "fs";

vi.mock('rotating-file-stream', () => {
    return {
        createStream: vi.fn(() => ({
            write: vi.fn(),
            end: vi.fn(),
            on: vi.fn(),
        })),
    }
})


describe("logger", () => {
    const OLD_ENV = process.env

    beforeEach(() => {
        process.env = {
            ...OLD_ENV,
            PORT: "8080",
            CONFIG_LOG: JSON.stringify({
                projectName: "test",
                namespace: "test",
                summary: {
                    console: true,
                    file: true,
                    path: "./logs/detail/",
                    rawData: true,
                    size: 100,
                },
                detail: {
                    console: true,
                    file: true,
                    path: "./logs/detail/",
                    rawData: true,
                    size: 100,
                },
            })
        }
    })

    afterEach(() => {
        vi.restoreAllMocks()
        vi.clearAllMocks()
        process.env = OLD_ENV
    })
    // test logConfig
    test("logConfig", () => {


        const existsSyncSp = vi.spyOn(fs, "existsSync").mockReturnValue(false)
        const mkdirSyncSp = vi.spyOn(fs, "mkdirSync").mockImplementation(() => (""))
        const conf: IConfig = {
            LogConfig: {
                namespace: "",
                detail: {
                    console: true,
                    file: true,
                    path: "./logs/detail/",
                    rawData: true,
                    size: 100,
                },
                summary: {
                    console: true,
                    file: true,
                    path: "./logs/detail/",
                    rawData: true,
                    size: 100,
                }
            }
        }

        loadLogConfig(conf)
        writeLogFile('dtl', 'test')

        expect(existsSyncSp).toHaveBeenCalledTimes(2)
        expect(mkdirSyncSp).toHaveBeenCalledTimes(2)
    })


})