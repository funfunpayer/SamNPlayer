export namespace generator {
	
	export class ProfileSuggestion {
	    Found: boolean;
	    Label: string;
	    Kind: string;
	    Confidence: number;
	
	    static createFrom(source: any = {}) {
	        return new ProfileSuggestion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Found = source["Found"];
	        this.Label = source["Label"];
	        this.Kind = source["Kind"];
	        this.Confidence = source["Confidence"];
	    }
	}

}

export namespace main {
	
	export class CacheInfo {
	    path: string;
	    files: number;
	    bytes: number;
	    humanSize: string;
	    clearOnExit: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CacheInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.files = source["files"];
	        this.bytes = source["bytes"];
	        this.humanSize = source["humanSize"];
	        this.clearOnExit = source["clearOnExit"];
	    }
	}
	export class CurvePoint {
	    atMs: number;
	    pos: number;
	
	    static createFrom(source: any = {}) {
	        return new CurvePoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.atMs = source["atMs"];
	        this.pos = source["pos"];
	    }
	}
	export class DeviceStatus {
	    connected: boolean;
	    mock: boolean;
	    name: string;
	    address: string;
	    rssi: number;
	    sessionActive: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DeviceStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connected = source["connected"];
	        this.mock = source["mock"];
	        this.name = source["name"];
	        this.address = source["address"];
	        this.rssi = source["rssi"];
	        this.sessionActive = source["sessionActive"];
	    }
	}
	export class FramePreview {
	    width: number;
	    height: number;
	    pngBase64: string;
	
	    static createFrom(source: any = {}) {
	        return new FramePreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.width = source["width"];
	        this.height = source["height"];
	        this.pngBase64 = source["pngBase64"];
	    }
	}
	export class GenerateOptions {
	    videoPath: string;
	    x: number;
	    y: number;
	    w: number;
	    h: number;
	    x2: number;
	    y2: number;
	    w2: number;
	    h2: number;
	    invert: boolean;
	    smoothWindow: number;
	    minPeakDistanceMs: number;
	    disableCameraCompensation: boolean;
	    disableSceneCutDetection: boolean;
	    rdpTolerance: number;
	    perSceneRoi: boolean;
	    adaptiveKeyframeError: number;
	    maxSpeed: number;
	    axis: string;
	    autoRetry: boolean;
	    backend: string;
	    useOpenCl: boolean;
	    dynamicRangeMs: number;
	    profile: string;
	    overwrite: boolean;
	    aiQualityOpinion: boolean;
	    contactVibration: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GenerateOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.videoPath = source["videoPath"];
	        this.x = source["x"];
	        this.y = source["y"];
	        this.w = source["w"];
	        this.h = source["h"];
	        this.x2 = source["x2"];
	        this.y2 = source["y2"];
	        this.w2 = source["w2"];
	        this.h2 = source["h2"];
	        this.invert = source["invert"];
	        this.smoothWindow = source["smoothWindow"];
	        this.minPeakDistanceMs = source["minPeakDistanceMs"];
	        this.disableCameraCompensation = source["disableCameraCompensation"];
	        this.disableSceneCutDetection = source["disableSceneCutDetection"];
	        this.rdpTolerance = source["rdpTolerance"];
	        this.perSceneRoi = source["perSceneRoi"];
	        this.adaptiveKeyframeError = source["adaptiveKeyframeError"];
	        this.maxSpeed = source["maxSpeed"];
	        this.axis = source["axis"];
	        this.autoRetry = source["autoRetry"];
	        this.backend = source["backend"];
	        this.useOpenCl = source["useOpenCl"];
	        this.dynamicRangeMs = source["dynamicRangeMs"];
	        this.profile = source["profile"];
	        this.overwrite = source["overwrite"];
	        this.aiQualityOpinion = source["aiQualityOpinion"];
	        this.contactVibration = source["contactVibration"];
	    }
	}
	export class HeatmapPoint {
	    atMs: number;
	    intensity: number;
	
	    static createFrom(source: any = {}) {
	        return new HeatmapPoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.atMs = source["atMs"];
	        this.intensity = source["intensity"];
	    }
	}
	export class Marker {
	    startMs: number;
	    endMs: number;
	
	    static createFrom(source: any = {}) {
	        return new Marker(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.startMs = source["startMs"];
	        this.endMs = source["endMs"];
	    }
	}
	export class PlaybackOptions {
	    mock: boolean;
	    syncMode: string;
	    tickMs: number;
	    maxSpeed: number;
	    smoothing: number;
	    softStartMs: number;
	    useVideoSync: boolean;
	    extendedOEnabled: boolean;
	    extendedOMin: number;
	    extendedOHoldS: number;
	    extendedORestoreMs: number;
	
	    static createFrom(source: any = {}) {
	        return new PlaybackOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mock = source["mock"];
	        this.syncMode = source["syncMode"];
	        this.tickMs = source["tickMs"];
	        this.maxSpeed = source["maxSpeed"];
	        this.smoothing = source["smoothing"];
	        this.softStartMs = source["softStartMs"];
	        this.useVideoSync = source["useVideoSync"];
	        this.extendedOEnabled = source["extendedOEnabled"];
	        this.extendedOMin = source["extendedOMin"];
	        this.extendedOHoldS = source["extendedOHoldS"];
	        this.extendedORestoreMs = source["extendedORestoreMs"];
	    }
	}
	export class ReportFeedback {
	    outputPath: string;
	    verdict: string;
	    comment: string;
	
	    static createFrom(source: any = {}) {
	        return new ReportFeedback(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.outputPath = source["outputPath"];
	        this.verdict = source["verdict"];
	        this.comment = source["comment"];
	    }
	}
	export class ScriptSegment {
	    state: string;
	    startMs: number;
	    endMs: number;
	    meanSpeed: number;
	
	    static createFrom(source: any = {}) {
	        return new ScriptSegment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.startMs = source["startMs"];
	        this.endMs = source["endMs"];
	        this.meanSpeed = source["meanSpeed"];
	    }
	}
	export class ScriptAnalysis {
	    segments: ScriptSegment[];
	    shareByState: Record<string, number>;
	    totalMs: number;
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new ScriptAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.segments = this.convertValues(source["segments"], ScriptSegment);
	        this.shareByState = source["shareByState"];
	        this.totalMs = source["totalMs"];
	        this.summary = source["summary"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ScriptInfo {
	    path: string;
	    actionCount: number;
	    durationMs: number;
	    videoPath: string;
	    hasVideo: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ScriptInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.actionCount = source["actionCount"];
	        this.durationMs = source["durationMs"];
	        this.videoPath = source["videoPath"];
	        this.hasVideo = source["hasVideo"];
	    }
	}
	
	export class Settings {
	    updateCheckOnStartup: boolean;
	    logLevel: string;
	    playbackMock: boolean;
	    playbackSync: string;
	    playbackTickMs: number;
	    playbackMaxSpeed: number;
	    playbackSmoothing: number;
	    playbackSoftStartMs: number;
	    playbackEOEnabled: boolean;
	    playbackEOMin: number;
	    playbackEOHoldS: number;
	    playbackEORestoreMs: number;
	    trainingMock: boolean;
	    trainingTechnique: string;
	    trainingChannel: string;
	    trainingCycles: number;
	    trainingRampUpMs: number;
	    trainingHoldMs: number;
	    trainingRestMs: number;
	    trainingPeakIntensity: number;
	    trainingPlateauFraction: number;
	    trainingProgressionPerCycle: number;
	    logPath: string;
	    reportPath: string;
	    defaultReportPath: string;
	    clearCacheOnExit: boolean;
	    deviceTransport: string;
	    intifaceUrl: string;
	    aiRoiModelPath: string;
	    aiBaseUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.updateCheckOnStartup = source["updateCheckOnStartup"];
	        this.logLevel = source["logLevel"];
	        this.playbackMock = source["playbackMock"];
	        this.playbackSync = source["playbackSync"];
	        this.playbackTickMs = source["playbackTickMs"];
	        this.playbackMaxSpeed = source["playbackMaxSpeed"];
	        this.playbackSmoothing = source["playbackSmoothing"];
	        this.playbackSoftStartMs = source["playbackSoftStartMs"];
	        this.playbackEOEnabled = source["playbackEOEnabled"];
	        this.playbackEOMin = source["playbackEOMin"];
	        this.playbackEOHoldS = source["playbackEOHoldS"];
	        this.playbackEORestoreMs = source["playbackEORestoreMs"];
	        this.trainingMock = source["trainingMock"];
	        this.trainingTechnique = source["trainingTechnique"];
	        this.trainingChannel = source["trainingChannel"];
	        this.trainingCycles = source["trainingCycles"];
	        this.trainingRampUpMs = source["trainingRampUpMs"];
	        this.trainingHoldMs = source["trainingHoldMs"];
	        this.trainingRestMs = source["trainingRestMs"];
	        this.trainingPeakIntensity = source["trainingPeakIntensity"];
	        this.trainingPlateauFraction = source["trainingPlateauFraction"];
	        this.trainingProgressionPerCycle = source["trainingProgressionPerCycle"];
	        this.logPath = source["logPath"];
	        this.reportPath = source["reportPath"];
	        this.defaultReportPath = source["defaultReportPath"];
	        this.clearCacheOnExit = source["clearCacheOnExit"];
	        this.deviceTransport = source["deviceTransport"];
	        this.intifaceUrl = source["intifaceUrl"];
	        this.aiRoiModelPath = source["aiRoiModelPath"];
	        this.aiBaseUrl = source["aiBaseUrl"];
	    }
	}
	export class TrainingRequest {
	    mock: boolean;
	    technique: string;
	    channel: string;
	    cycles: number;
	    rampUpMs: number;
	    holdMs: number;
	    restMs: number;
	    peakIntensity: number;
	    plateauFraction: number;
	    progressionPerCycle: number;
	
	    static createFrom(source: any = {}) {
	        return new TrainingRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mock = source["mock"];
	        this.technique = source["technique"];
	        this.channel = source["channel"];
	        this.cycles = source["cycles"];
	        this.rampUpMs = source["rampUpMs"];
	        this.holdMs = source["holdMs"];
	        this.restMs = source["restMs"];
	        this.peakIntensity = source["peakIntensity"];
	        this.plateauFraction = source["plateauFraction"];
	        this.progressionPerCycle = source["progressionPerCycle"];
	    }
	}
	export class UpdateCheckResult {
	    available: boolean;
	    release?: update.Release;
	    assetName?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.release = this.convertValues(source["release"], update.Release);
	        this.assetName = source["assetName"];
	        this.error = source["error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace update {
	
	export class Asset {
	    name: string;
	    browser_download_url: string;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new Asset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.browser_download_url = source["browser_download_url"];
	        this.size = source["size"];
	    }
	}
	export class Release {
	    tag_name: string;
	    html_url: string;
	    assets: Asset[];
	
	    static createFrom(source: any = {}) {
	        return new Release(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tag_name = source["tag_name"];
	        this.html_url = source["html_url"];
	        this.assets = this.convertValues(source["assets"], Asset);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

