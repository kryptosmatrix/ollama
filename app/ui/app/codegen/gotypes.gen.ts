export type JSONValue = string | number | boolean | null | JSONValue[] | { [key: string]: JSONValue };

/* Do not change, this code is generated from Golang structs */


export class ChatInfo {
    id: string;
    title: string;
    userExcerpt: string;
    createdAt: Date;
    updatedAt: Date;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.id = (source as Record<string, unknown>)["id"] as ChatInfo["id"];
        this.title = (source as Record<string, unknown>)["title"] as ChatInfo["title"];
        this.userExcerpt = (source as Record<string, unknown>)["userExcerpt"] as ChatInfo["userExcerpt"];
        this.createdAt = new Date((source as Record<string, unknown>)["createdAt"] as string | number | Date) as ChatInfo["createdAt"];
        this.updatedAt = new Date((source as Record<string, unknown>)["updatedAt"] as string | number | Date) as ChatInfo["updatedAt"];
    }
}
export class ChatsResponse {
    chatInfos: ChatInfo[];

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.chatInfos = this.convertValues((source as Record<string, unknown>)["chatInfos"], ChatInfo) as ChatsResponse["chatInfos"];
    }

	convertValues(a: unknown, classs: new (source: unknown) => unknown, asMap: boolean = false): unknown {
	    if (!a) {
	        return a;
	    }
	    if (Array.isArray(a)) {
	        return (a as unknown[]).map(elem => this.convertValues(elem, classs));
	    } else if ("object" === typeof a) {
	        if (asMap) {
	            for (const key of Object.keys(a)) {
	                (a as Record<string, unknown>)[key] = new classs((a as Record<string, unknown>)[key]);
	            }
	            return a;
	        }
	        return new classs(a);
	    }
	    return a;
	}
}
export class Time {


    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);

    }
}
export class ToolFunction {
    name: string;
    arguments: string;
    result?: JSONValue;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.name = (source as Record<string, unknown>)["name"] as ToolFunction["name"];
        this.arguments = (source as Record<string, unknown>)["arguments"] as ToolFunction["arguments"];
        this.result = (source as Record<string, unknown>)["result"] as ToolFunction["result"];
    }
}
export class ToolCall {
    type: string;
    function: ToolFunction;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.type = (source as Record<string, unknown>)["type"] as ToolCall["type"];
        this.function = this.convertValues((source as Record<string, unknown>)["function"], ToolFunction) as ToolCall["function"];
    }

	convertValues(a: unknown, classs: new (source: unknown) => unknown, asMap: boolean = false): unknown {
	    if (!a) {
	        return a;
	    }
	    if (Array.isArray(a)) {
	        return (a as unknown[]).map(elem => this.convertValues(elem, classs));
	    } else if ("object" === typeof a) {
	        if (asMap) {
	            for (const key of Object.keys(a)) {
	                (a as Record<string, unknown>)[key] = new classs((a as Record<string, unknown>)[key]);
	            }
	            return a;
	        }
	        return new classs(a);
	    }
	    return a;
	}
}
export class File {
    filename: string;
    data: number[];

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.filename = (source as Record<string, unknown>)["filename"] as File["filename"];
        this.data = (source as Record<string, unknown>)["data"] as File["data"];
    }
}
export class Message {
    role: string;
    content: string;
    thinking: string;
    stream: boolean;
    model?: string;
    attachments?: File[];
    tool_calls?: ToolCall[];
    tool_call?: ToolCall;
    tool_name?: string;
    tool_result?: number[];
    created_at: Time;
    updated_at: Time;
    thinkingTimeStart?: Date | undefined;
    thinkingTimeEnd?: Date | undefined;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.role = (source as Record<string, unknown>)["role"] as Message["role"];
        this.content = (source as Record<string, unknown>)["content"] as Message["content"];
        this.thinking = (source as Record<string, unknown>)["thinking"] as Message["thinking"];
        this.stream = (source as Record<string, unknown>)["stream"] as Message["stream"];
        this.model = (source as Record<string, unknown>)["model"] as Message["model"];
        this.attachments = this.convertValues((source as Record<string, unknown>)["attachments"], File) as Message["attachments"];
        this.tool_calls = this.convertValues((source as Record<string, unknown>)["tool_calls"], ToolCall) as Message["tool_calls"];
        this.tool_call = this.convertValues((source as Record<string, unknown>)["tool_call"], ToolCall) as Message["tool_call"];
        this.tool_name = (source as Record<string, unknown>)["tool_name"] as Message["tool_name"];
        this.tool_result = (source as Record<string, unknown>)["tool_result"] as Message["tool_result"];
        this.created_at = this.convertValues((source as Record<string, unknown>)["created_at"], Time) as Message["created_at"];
        this.updated_at = this.convertValues((source as Record<string, unknown>)["updated_at"], Time) as Message["updated_at"];
        this.thinkingTimeStart = (source as Record<string, unknown>)["thinkingTimeStart"] as Message["thinkingTimeStart"] && new Date((source as Record<string, unknown>)["thinkingTimeStart"] as string | number | Date);
        this.thinkingTimeEnd = (source as Record<string, unknown>)["thinkingTimeEnd"] as Message["thinkingTimeEnd"] && new Date((source as Record<string, unknown>)["thinkingTimeEnd"] as string | number | Date);
    }

	convertValues(a: unknown, classs: new (source: unknown) => unknown, asMap: boolean = false): unknown {
	    if (!a) {
	        return a;
	    }
	    if (Array.isArray(a)) {
	        return (a as unknown[]).map(elem => this.convertValues(elem, classs));
	    } else if ("object" === typeof a) {
	        if (asMap) {
	            for (const key of Object.keys(a)) {
	                (a as Record<string, unknown>)[key] = new classs((a as Record<string, unknown>)[key]);
	            }
	            return a;
	        }
	        return new classs(a);
	    }
	    return a;
	}
}
export class Chat {
    id: string;
    messages: Message[];
    title: string;
    created_at: Time;
    browser_state?: BrowserStateData;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.id = (source as Record<string, unknown>)["id"] as Chat["id"];
        this.messages = this.convertValues((source as Record<string, unknown>)["messages"], Message) as Chat["messages"];
        this.title = (source as Record<string, unknown>)["title"] as Chat["title"];
        this.created_at = this.convertValues((source as Record<string, unknown>)["created_at"], Time) as Chat["created_at"];
        this.browser_state = (source as Record<string, unknown>)["browser_state"] as Chat["browser_state"];
    }

	convertValues(a: unknown, classs: new (source: unknown) => unknown, asMap: boolean = false): unknown {
	    if (!a) {
	        return a;
	    }
	    if (Array.isArray(a)) {
	        return (a as unknown[]).map(elem => this.convertValues(elem, classs));
	    } else if ("object" === typeof a) {
	        if (asMap) {
	            for (const key of Object.keys(a)) {
	                (a as Record<string, unknown>)[key] = new classs((a as Record<string, unknown>)[key]);
	            }
	            return a;
	        }
	        return new classs(a);
	    }
	    return a;
	}
}
export class ChatResponse {
    chat: Chat;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.chat = this.convertValues((source as Record<string, unknown>)["chat"], Chat) as ChatResponse["chat"];
    }

	convertValues(a: unknown, classs: new (source: unknown) => unknown, asMap: boolean = false): unknown {
	    if (!a) {
	        return a;
	    }
	    if (Array.isArray(a)) {
	        return (a as unknown[]).map(elem => this.convertValues(elem, classs));
	    } else if ("object" === typeof a) {
	        if (asMap) {
	            for (const key of Object.keys(a)) {
	                (a as Record<string, unknown>)[key] = new classs((a as Record<string, unknown>)[key]);
	            }
	            return a;
	        }
	        return new classs(a);
	    }
	    return a;
	}
}
export class Model {
    model: string;
    digest?: string;
    modified_at?: Time;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.model = (source as Record<string, unknown>)["model"] as Model["model"];
        this.digest = (source as Record<string, unknown>)["digest"] as Model["digest"];
        this.modified_at = this.convertValues((source as Record<string, unknown>)["modified_at"], Time) as Model["modified_at"];
    }

	convertValues(a: unknown, classs: new (source: unknown) => unknown, asMap: boolean = false): unknown {
	    if (!a) {
	        return a;
	    }
	    if (Array.isArray(a)) {
	        return (a as unknown[]).map(elem => this.convertValues(elem, classs));
	    } else if ("object" === typeof a) {
	        if (asMap) {
	            for (const key of Object.keys(a)) {
	                (a as Record<string, unknown>)[key] = new classs((a as Record<string, unknown>)[key]);
	            }
	            return a;
	        }
	        return new classs(a);
	    }
	    return a;
	}
}
export class ModelsResponse {
    models: Model[];

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.models = this.convertValues((source as Record<string, unknown>)["models"], Model) as ModelsResponse["models"];
    }

	convertValues(a: unknown, classs: new (source: unknown) => unknown, asMap: boolean = false): unknown {
	    if (!a) {
	        return a;
	    }
	    if (Array.isArray(a)) {
	        return (a as unknown[]).map(elem => this.convertValues(elem, classs));
	    } else if ("object" === typeof a) {
	        if (asMap) {
	            for (const key of Object.keys(a)) {
	                (a as Record<string, unknown>)[key] = new classs((a as Record<string, unknown>)[key]);
	            }
	            return a;
	        }
	        return new classs(a);
	    }
	    return a;
	}
}
export class InferenceCompute {
    library: string;
    variant: string;
    compute: string;
    driver: string;
    name: string;
    vram: string;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.library = (source as Record<string, unknown>)["library"] as InferenceCompute["library"];
        this.variant = (source as Record<string, unknown>)["variant"] as InferenceCompute["variant"];
        this.compute = (source as Record<string, unknown>)["compute"] as InferenceCompute["compute"];
        this.driver = (source as Record<string, unknown>)["driver"] as InferenceCompute["driver"];
        this.name = (source as Record<string, unknown>)["name"] as InferenceCompute["name"];
        this.vram = (source as Record<string, unknown>)["vram"] as InferenceCompute["vram"];
    }
}
export class InferenceComputeResponse {
    inferenceComputes: InferenceCompute[];
    defaultContextLength: number;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.inferenceComputes = this.convertValues((source as Record<string, unknown>)["inferenceComputes"], InferenceCompute) as InferenceComputeResponse["inferenceComputes"];
        this.defaultContextLength = (source as Record<string, unknown>)["defaultContextLength"] as InferenceComputeResponse["defaultContextLength"];
    }

	convertValues(a: unknown, classs: new (source: unknown) => unknown, asMap: boolean = false): unknown {
	    if (!a) {
	        return a;
	    }
	    if (Array.isArray(a)) {
	        return (a as unknown[]).map(elem => this.convertValues(elem, classs));
	    } else if ("object" === typeof a) {
	        if (asMap) {
	            for (const key of Object.keys(a)) {
	                (a as Record<string, unknown>)[key] = new classs((a as Record<string, unknown>)[key]);
	            }
	            return a;
	        }
	        return new classs(a);
	    }
	    return a;
	}
}
export class ModelCapabilitiesResponse {
    capabilities: string[];

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.capabilities = (source as Record<string, unknown>)["capabilities"] as ModelCapabilitiesResponse["capabilities"];
    }
}
export class ChatEvent {
    eventName: "chat" | "thinking" | "assistant_with_tools" | "tool_call" | "tool" | "tool_result" | "tool_approval" | "done" | "chat_created";
    content?: string;
    thinking?: string;
    thinkingTimeStart?: Date | undefined;
    thinkingTimeEnd?: Date | undefined;
    toolCalls?: ToolCall[];
    toolCall?: ToolCall;
    toolName?: string;
    toolResult?: boolean;
    toolResultData?: JSONValue;
    chatId?: string;
    approvalId?: string;
    approvalScope?: string;
    approvalArgs?: {[key: string]: JSONValue};
    toolState?: JSONValue;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.eventName = (source as Record<string, unknown>)["eventName"] as ChatEvent["eventName"];
        this.content = (source as Record<string, unknown>)["content"] as ChatEvent["content"];
        this.thinking = (source as Record<string, unknown>)["thinking"] as ChatEvent["thinking"];
        this.thinkingTimeStart = (source as Record<string, unknown>)["thinkingTimeStart"] as ChatEvent["thinkingTimeStart"] && new Date((source as Record<string, unknown>)["thinkingTimeStart"] as string | number | Date);
        this.thinkingTimeEnd = (source as Record<string, unknown>)["thinkingTimeEnd"] as ChatEvent["thinkingTimeEnd"] && new Date((source as Record<string, unknown>)["thinkingTimeEnd"] as string | number | Date);
        this.toolCalls = this.convertValues((source as Record<string, unknown>)["toolCalls"], ToolCall) as ChatEvent["toolCalls"];
        this.toolCall = this.convertValues((source as Record<string, unknown>)["toolCall"], ToolCall) as ChatEvent["toolCall"];
        this.toolName = (source as Record<string, unknown>)["toolName"] as ChatEvent["toolName"];
        this.toolResult = (source as Record<string, unknown>)["toolResult"] as ChatEvent["toolResult"];
        this.toolResultData = (source as Record<string, unknown>)["toolResultData"] as ChatEvent["toolResultData"];
        this.chatId = (source as Record<string, unknown>)["chatId"] as ChatEvent["chatId"];
        this.approvalId = (source as Record<string, unknown>)["approvalId"] as ChatEvent["approvalId"];
        this.approvalScope = (source as Record<string, unknown>)["approvalScope"] as ChatEvent["approvalScope"];
        this.approvalArgs = (source as Record<string, unknown>)["approvalArgs"] as ChatEvent["approvalArgs"];
        this.toolState = (source as Record<string, unknown>)["toolState"] as ChatEvent["toolState"];
    }

	convertValues(a: unknown, classs: new (source: unknown) => unknown, asMap: boolean = false): unknown {
	    if (!a) {
	        return a;
	    }
	    if (Array.isArray(a)) {
	        return (a as unknown[]).map(elem => this.convertValues(elem, classs));
	    } else if ("object" === typeof a) {
	        if (asMap) {
	            for (const key of Object.keys(a)) {
	                (a as Record<string, unknown>)[key] = new classs((a as Record<string, unknown>)[key]);
	            }
	            return a;
	        }
	        return new classs(a);
	    }
	    return a;
	}
}
export class DownloadEvent {
    eventName: "download";
    total: number;
    completed: number;
    done: boolean;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.eventName = (source as Record<string, unknown>)["eventName"] as DownloadEvent["eventName"];
        this.total = (source as Record<string, unknown>)["total"] as DownloadEvent["total"];
        this.completed = (source as Record<string, unknown>)["completed"] as DownloadEvent["completed"];
        this.done = (source as Record<string, unknown>)["done"] as DownloadEvent["done"];
    }
}
export class ErrorEvent {
    eventName: "error";
    error: string;
    code?: string;
    details?: string;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.eventName = (source as Record<string, unknown>)["eventName"] as ErrorEvent["eventName"];
        this.error = (source as Record<string, unknown>)["error"] as ErrorEvent["error"];
        this.code = (source as Record<string, unknown>)["code"] as ErrorEvent["code"];
        this.details = (source as Record<string, unknown>)["details"] as ErrorEvent["details"];
    }
}
export class Settings {
    Expose: boolean;
    Browser: boolean;
    Survey: boolean;
    Models: string;
    Agent: boolean;
    Tools: boolean;
    WorkingDir: string;
    ContextLength: number;
    TurboEnabled: boolean;
    WebSearchEnabled: boolean;
    ThinkEnabled: boolean;
    ThinkLevel: string;
    SelectedModel: string;
    SidebarOpen: boolean;
    LastHomeView: string;
    AutoUpdateEnabled: boolean;
    AutoApproveTools: boolean;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.Expose = (source as Record<string, unknown>)["Expose"] as Settings["Expose"];
        this.Browser = (source as Record<string, unknown>)["Browser"] as Settings["Browser"];
        this.Survey = (source as Record<string, unknown>)["Survey"] as Settings["Survey"];
        this.Models = (source as Record<string, unknown>)["Models"] as Settings["Models"];
        this.Agent = (source as Record<string, unknown>)["Agent"] as Settings["Agent"];
        this.Tools = (source as Record<string, unknown>)["Tools"] as Settings["Tools"];
        this.WorkingDir = (source as Record<string, unknown>)["WorkingDir"] as Settings["WorkingDir"];
        this.ContextLength = (source as Record<string, unknown>)["ContextLength"] as Settings["ContextLength"];
        this.TurboEnabled = (source as Record<string, unknown>)["TurboEnabled"] as Settings["TurboEnabled"];
        this.WebSearchEnabled = (source as Record<string, unknown>)["WebSearchEnabled"] as Settings["WebSearchEnabled"];
        this.ThinkEnabled = (source as Record<string, unknown>)["ThinkEnabled"] as Settings["ThinkEnabled"];
        this.ThinkLevel = (source as Record<string, unknown>)["ThinkLevel"] as Settings["ThinkLevel"];
        this.SelectedModel = (source as Record<string, unknown>)["SelectedModel"] as Settings["SelectedModel"];
        this.SidebarOpen = (source as Record<string, unknown>)["SidebarOpen"] as Settings["SidebarOpen"];
        this.LastHomeView = (source as Record<string, unknown>)["LastHomeView"] as Settings["LastHomeView"];
        this.AutoUpdateEnabled = (source as Record<string, unknown>)["AutoUpdateEnabled"] as Settings["AutoUpdateEnabled"];
        this.AutoApproveTools = (source as Record<string, unknown>)["AutoApproveTools"] as Settings["AutoApproveTools"];
    }
}
export class SettingsResponse {
    settings: Settings;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.settings = this.convertValues((source as Record<string, unknown>)["settings"], Settings) as SettingsResponse["settings"];
    }

	convertValues(a: unknown, classs: new (source: unknown) => unknown, asMap: boolean = false): unknown {
	    if (!a) {
	        return a;
	    }
	    if (Array.isArray(a)) {
	        return (a as unknown[]).map(elem => this.convertValues(elem, classs));
	    } else if ("object" === typeof a) {
	        if (asMap) {
	            for (const key of Object.keys(a)) {
	                (a as Record<string, unknown>)[key] = new classs((a as Record<string, unknown>)[key]);
	            }
	            return a;
	        }
	        return new classs(a);
	    }
	    return a;
	}
}
export class HealthResponse {
    healthy: boolean;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.healthy = (source as Record<string, unknown>)["healthy"] as HealthResponse["healthy"];
    }
}
export class User {
    id: string;
    email: string;
    name: string;
    bio?: string;
    avatarurl?: string;
    firstname?: string;
    lastname?: string;
    plan?: string;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.id = (source as Record<string, unknown>)["id"] as User["id"];
        this.email = (source as Record<string, unknown>)["email"] as User["email"];
        this.name = (source as Record<string, unknown>)["name"] as User["name"];
        this.bio = (source as Record<string, unknown>)["bio"] as User["bio"];
        this.avatarurl = (source as Record<string, unknown>)["avatarurl"] as User["avatarurl"];
        this.firstname = (source as Record<string, unknown>)["firstname"] as User["firstname"];
        this.lastname = (source as Record<string, unknown>)["lastname"] as User["lastname"];
        this.plan = (source as Record<string, unknown>)["plan"] as User["plan"];
    }
}
export class Attachment {
    filename: string;
    data?: string;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.filename = (source as Record<string, unknown>)["filename"] as Attachment["filename"];
        this.data = (source as Record<string, unknown>)["data"] as Attachment["data"];
    }
}
export class ChatRequest {
    model: string;
    prompt: string;
    index?: number;
    attachments?: Attachment[];
    web_search?: boolean;
    file_tools?: boolean;
    forceUpdate?: boolean;
    think?: JSONValue;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.model = (source as Record<string, unknown>)["model"] as ChatRequest["model"];
        this.prompt = (source as Record<string, unknown>)["prompt"] as ChatRequest["prompt"];
        this.index = (source as Record<string, unknown>)["index"] as ChatRequest["index"];
        this.attachments = this.convertValues((source as Record<string, unknown>)["attachments"], Attachment) as ChatRequest["attachments"];
        this.web_search = (source as Record<string, unknown>)["web_search"] as ChatRequest["web_search"];
        this.file_tools = (source as Record<string, unknown>)["file_tools"] as ChatRequest["file_tools"];
        this.forceUpdate = (source as Record<string, unknown>)["forceUpdate"] as ChatRequest["forceUpdate"];
        this.think = (source as Record<string, unknown>)["think"] as ChatRequest["think"];
    }

	convertValues(a: unknown, classs: new (source: unknown) => unknown, asMap: boolean = false): unknown {
	    if (!a) {
	        return a;
	    }
	    if (Array.isArray(a)) {
	        return (a as unknown[]).map(elem => this.convertValues(elem, classs));
	    } else if ("object" === typeof a) {
	        if (asMap) {
	            for (const key of Object.keys(a)) {
	                (a as Record<string, unknown>)[key] = new classs((a as Record<string, unknown>)[key]);
	            }
	            return a;
	        }
	        return new classs(a);
	    }
	    return a;
	}
}
export class Error {
    error: string;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.error = (source as Record<string, unknown>)["error"] as Error["error"];
    }
}
export class ModelUpstreamResponse {
    stale: boolean;
    error?: string;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.stale = (source as Record<string, unknown>)["stale"] as ModelUpstreamResponse["stale"];
        this.error = (source as Record<string, unknown>)["error"] as ModelUpstreamResponse["error"];
    }
}
export class Page {
    url: string;
    title: string;
    text: string;
    lines: string[];
    links?: Record<number, string>;
    fetched_at: Time;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.url = (source as Record<string, unknown>)["url"] as Page["url"];
        this.title = (source as Record<string, unknown>)["title"] as Page["title"];
        this.text = (source as Record<string, unknown>)["text"] as Page["text"];
        this.lines = (source as Record<string, unknown>)["lines"] as Page["lines"];
        this.links = (source as Record<string, unknown>)["links"] as Page["links"];
        this.fetched_at = this.convertValues((source as Record<string, unknown>)["fetched_at"], Time) as Page["fetched_at"];
    }

	convertValues(a: unknown, classs: new (source: unknown) => unknown, asMap: boolean = false): unknown {
	    if (!a) {
	        return a;
	    }
	    if (Array.isArray(a)) {
	        return (a as unknown[]).map(elem => this.convertValues(elem, classs));
	    } else if ("object" === typeof a) {
	        if (asMap) {
	            for (const key of Object.keys(a)) {
	                (a as Record<string, unknown>)[key] = new classs((a as Record<string, unknown>)[key]);
	            }
	            return a;
	        }
	        return new classs(a);
	    }
	    return a;
	}
}
export class BrowserStateData {
    page_stack: string[];
    view_tokens: number;
    url_to_page: {[key: string]: Page};

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.page_stack = (source as Record<string, unknown>)["page_stack"] as BrowserStateData["page_stack"];
        this.view_tokens = (source as Record<string, unknown>)["view_tokens"] as BrowserStateData["view_tokens"];
        this.url_to_page = (source as Record<string, unknown>)["url_to_page"] as BrowserStateData["url_to_page"];
    }
}

export class MCPTool {
    name: string;
    description: string;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.name = (source as Record<string, unknown>)["name"] as MCPTool["name"];
        this.description = (source as Record<string, unknown>)["description"] as MCPTool["description"];
    }
}
export class MCPSkippedTool {
    name: string;
    reason: string;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.name = (source as Record<string, unknown>)["name"] as MCPSkippedTool["name"];
        this.reason = (source as Record<string, unknown>)["reason"] as MCPSkippedTool["reason"];
    }
}
export class MCPServer {
    name: string;
    status: string;
    transport: string;
    runs: string;
    enabled: boolean;
    approved: boolean;
    changed?: boolean;
    previouslyRan?: string;
    error?: string;
    tools?: MCPTool[];
    skipped?: MCPSkippedTool[];
    canSignIn?: boolean;
    signedIn?: boolean;
    signingIn?: boolean;
    warnings?: string[];
    tokenStore?: string;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.name = (source as Record<string, unknown>)["name"] as MCPServer["name"];
        this.status = (source as Record<string, unknown>)["status"] as MCPServer["status"];
        this.transport = (source as Record<string, unknown>)["transport"] as MCPServer["transport"];
        this.runs = (source as Record<string, unknown>)["runs"] as MCPServer["runs"];
        this.enabled = (source as Record<string, unknown>)["enabled"] as MCPServer["enabled"];
        this.approved = (source as Record<string, unknown>)["approved"] as MCPServer["approved"];
        this.changed = (source as Record<string, unknown>)["changed"] as MCPServer["changed"];
        this.previouslyRan = (source as Record<string, unknown>)["previouslyRan"] as MCPServer["previouslyRan"];
        this.error = (source as Record<string, unknown>)["error"] as MCPServer["error"];
        this.tools = this.convertValues((source as Record<string, unknown>)["tools"], MCPTool) as MCPServer["tools"];
        this.skipped = this.convertValues((source as Record<string, unknown>)["skipped"], MCPSkippedTool) as MCPServer["skipped"];
        this.canSignIn = (source as Record<string, unknown>)["canSignIn"] as MCPServer["canSignIn"];
        this.signedIn = (source as Record<string, unknown>)["signedIn"] as MCPServer["signedIn"];
        this.signingIn = (source as Record<string, unknown>)["signingIn"] as MCPServer["signingIn"];
        this.warnings = (source as Record<string, unknown>)["warnings"] as MCPServer["warnings"];
        this.tokenStore = (source as Record<string, unknown>)["tokenStore"] as MCPServer["tokenStore"];
    }

	convertValues(a: unknown, classs: new (source: unknown) => unknown, asMap: boolean = false): unknown {
	    if (!a) {
	        return a;
	    }
	    if (Array.isArray(a)) {
	        return (a as unknown[]).map(elem => this.convertValues(elem, classs));
	    } else if ("object" === typeof a) {
	        if (asMap) {
	            for (const key of Object.keys(a)) {
	                (a as Record<string, unknown>)[key] = new classs((a as Record<string, unknown>)[key]);
	            }
	            return a;
	        }
	        return new classs(a);
	    }
	    return a;
	}
}
export class MCPServersResponse {
    servers: MCPServer[];

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.servers = this.convertValues((source as Record<string, unknown>)["servers"], MCPServer) as MCPServersResponse["servers"];
    }

	convertValues(a: unknown, classs: new (source: unknown) => unknown, asMap: boolean = false): unknown {
	    if (!a) {
	        return a;
	    }
	    if (Array.isArray(a)) {
	        return (a as unknown[]).map(elem => this.convertValues(elem, classs));
	    } else if ("object" === typeof a) {
	        if (asMap) {
	            for (const key of Object.keys(a)) {
	                (a as Record<string, unknown>)[key] = new classs((a as Record<string, unknown>)[key]);
	            }
	            return a;
	        }
	        return new classs(a);
	    }
	    return a;
	}
}
export class MCPRegistryEntry {
    name: string;
    title?: string;
    description?: string;
    version?: string;
    publisher: string;
    repository?: string;
    websiteUrl?: string;
    installable: boolean;
    reason?: string;
    transport?: string;
    runs?: string;
    suggestedName?: string;
    command?: string;
    args?: string[];
    env?: {[key: string]: string};
    url?: string;
    headers?: {[key: string]: string};
    variables?: string[];

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.name = (source as Record<string, unknown>)["name"] as MCPRegistryEntry["name"];
        this.title = (source as Record<string, unknown>)["title"] as MCPRegistryEntry["title"];
        this.description = (source as Record<string, unknown>)["description"] as MCPRegistryEntry["description"];
        this.version = (source as Record<string, unknown>)["version"] as MCPRegistryEntry["version"];
        this.publisher = (source as Record<string, unknown>)["publisher"] as MCPRegistryEntry["publisher"];
        this.repository = (source as Record<string, unknown>)["repository"] as MCPRegistryEntry["repository"];
        this.websiteUrl = (source as Record<string, unknown>)["websiteUrl"] as MCPRegistryEntry["websiteUrl"];
        this.installable = (source as Record<string, unknown>)["installable"] as MCPRegistryEntry["installable"];
        this.reason = (source as Record<string, unknown>)["reason"] as MCPRegistryEntry["reason"];
        this.transport = (source as Record<string, unknown>)["transport"] as MCPRegistryEntry["transport"];
        this.runs = (source as Record<string, unknown>)["runs"] as MCPRegistryEntry["runs"];
        this.suggestedName = (source as Record<string, unknown>)["suggestedName"] as MCPRegistryEntry["suggestedName"];
        this.command = (source as Record<string, unknown>)["command"] as MCPRegistryEntry["command"];
        this.args = (source as Record<string, unknown>)["args"] as MCPRegistryEntry["args"];
        this.env = (source as Record<string, unknown>)["env"] as MCPRegistryEntry["env"];
        this.url = (source as Record<string, unknown>)["url"] as MCPRegistryEntry["url"];
        this.headers = (source as Record<string, unknown>)["headers"] as MCPRegistryEntry["headers"];
        this.variables = (source as Record<string, unknown>)["variables"] as MCPRegistryEntry["variables"];
    }
}
export class MCPRegistryResponse {
    entries: MCPRegistryEntry[];
    nextCursor?: string;
    notVetted: boolean;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.entries = this.convertValues((source as Record<string, unknown>)["entries"], MCPRegistryEntry) as MCPRegistryResponse["entries"];
        this.nextCursor = (source as Record<string, unknown>)["nextCursor"] as MCPRegistryResponse["nextCursor"];
        this.notVetted = (source as Record<string, unknown>)["notVetted"] as MCPRegistryResponse["notVetted"];
    }

	convertValues(a: unknown, classs: new (source: unknown) => unknown, asMap: boolean = false): unknown {
	    if (!a) {
	        return a;
	    }
	    if (Array.isArray(a)) {
	        return (a as unknown[]).map(elem => this.convertValues(elem, classs));
	    } else if ("object" === typeof a) {
	        if (asMap) {
	            for (const key of Object.keys(a)) {
	                (a as Record<string, unknown>)[key] = new classs((a as Record<string, unknown>)[key]);
	            }
	            return a;
	        }
	        return new classs(a);
	    }
	    return a;
	}
}
export class MCPDiscoveredServer {
    name: string;
    sources: string[];
    paths?: string[];
    runs: string;
    origin: string;
    notes?: string[];
    problem?: string;
    command?: string;
    args?: string[];
    env?: {[key: string]: string};
    url?: string;
    headers?: {[key: string]: string};
    disabled?: boolean;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.name = (source as Record<string, unknown>)["name"] as MCPDiscoveredServer["name"];
        this.sources = (source as Record<string, unknown>)["sources"] as MCPDiscoveredServer["sources"];
        this.paths = (source as Record<string, unknown>)["paths"] as MCPDiscoveredServer["paths"];
        this.runs = (source as Record<string, unknown>)["runs"] as MCPDiscoveredServer["runs"];
        this.origin = (source as Record<string, unknown>)["origin"] as MCPDiscoveredServer["origin"];
        this.notes = (source as Record<string, unknown>)["notes"] as MCPDiscoveredServer["notes"];
        this.problem = (source as Record<string, unknown>)["problem"] as MCPDiscoveredServer["problem"];
        this.command = (source as Record<string, unknown>)["command"] as MCPDiscoveredServer["command"];
        this.args = (source as Record<string, unknown>)["args"] as MCPDiscoveredServer["args"];
        this.env = (source as Record<string, unknown>)["env"] as MCPDiscoveredServer["env"];
        this.url = (source as Record<string, unknown>)["url"] as MCPDiscoveredServer["url"];
        this.headers = (source as Record<string, unknown>)["headers"] as MCPDiscoveredServer["headers"];
        this.disabled = (source as Record<string, unknown>)["disabled"] as MCPDiscoveredServer["disabled"];
    }
}
export class MCPDiscoveryResponse {
    servers: MCPDiscoveredServer[];
    searched?: string[];
    error?: string;

    constructor(source: unknown = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.servers = this.convertValues((source as Record<string, unknown>)["servers"], MCPDiscoveredServer) as MCPDiscoveryResponse["servers"];
        this.searched = (source as Record<string, unknown>)["searched"] as MCPDiscoveryResponse["searched"];
        this.error = (source as Record<string, unknown>)["error"] as MCPDiscoveryResponse["error"];
    }

	convertValues(a: unknown, classs: new (source: unknown) => unknown, asMap: boolean = false): unknown {
	    if (!a) {
	        return a;
	    }
	    if (Array.isArray(a)) {
	        return (a as unknown[]).map(elem => this.convertValues(elem, classs));
	    } else if ("object" === typeof a) {
	        if (asMap) {
	            for (const key of Object.keys(a)) {
	                (a as Record<string, unknown>)[key] = new classs((a as Record<string, unknown>)[key]);
	            }
	            return a;
	        }
	        return new classs(a);
	    }
	    return a;
	}
}