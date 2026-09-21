export class ChatInfo {
    constructor(source = {}) {
        Object.defineProperty(this, "id", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "title", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "userExcerpt", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "createdAt", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "updatedAt", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.id = source["id"];
        this.title = source["title"];
        this.userExcerpt = source["userExcerpt"];
        this.createdAt = new Date(source["createdAt"]);
        this.updatedAt = new Date(source["updatedAt"]);
    }
}
export class ChatsResponse {
    constructor(source = {}) {
        Object.defineProperty(this, "chatInfos", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.chatInfos = this.convertValues(source["chatInfos"], ChatInfo);
    }
    convertValues(a, classs, asMap = false) {
        if (!a) {
            return a;
        }
        if (Array.isArray(a)) {
            return a.map(elem => this.convertValues(elem, classs));
        }
        else if ("object" === typeof a) {
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
export class Time {
    constructor(source = {}) {
        if ('string' === typeof source)
            source = JSON.parse(source);
    }
}
export class ToolFunction {
    constructor(source = {}) {
        Object.defineProperty(this, "name", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "arguments", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "result", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.name = source["name"];
        this.arguments = source["arguments"];
        this.result = source["result"];
    }
}
export class ToolCall {
    constructor(source = {}) {
        Object.defineProperty(this, "type", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "function", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.type = source["type"];
        this.function = this.convertValues(source["function"], ToolFunction);
    }
    convertValues(a, classs, asMap = false) {
        if (!a) {
            return a;
        }
        if (Array.isArray(a)) {
            return a.map(elem => this.convertValues(elem, classs));
        }
        else if ("object" === typeof a) {
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
export class File {
    constructor(source = {}) {
        Object.defineProperty(this, "filename", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "data", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.filename = source["filename"];
        this.data = source["data"];
    }
}
export class Message {
    constructor(source = {}) {
        Object.defineProperty(this, "role", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "content", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "thinking", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "stream", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "model", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "attachments", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "tool_calls", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "tool_call", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "tool_name", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "tool_result", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "created_at", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "updated_at", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "thinkingTimeStart", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "thinkingTimeEnd", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.role = source["role"];
        this.content = source["content"];
        this.thinking = source["thinking"];
        this.stream = source["stream"];
        this.model = source["model"];
        this.attachments = this.convertValues(source["attachments"], File);
        this.tool_calls = this.convertValues(source["tool_calls"], ToolCall);
        this.tool_call = this.convertValues(source["tool_call"], ToolCall);
        this.tool_name = source["tool_name"];
        this.tool_result = source["tool_result"];
        this.created_at = this.convertValues(source["created_at"], Time);
        this.updated_at = this.convertValues(source["updated_at"], Time);
        this.thinkingTimeStart = source["thinkingTimeStart"] && new Date(source["thinkingTimeStart"]);
        this.thinkingTimeEnd = source["thinkingTimeEnd"] && new Date(source["thinkingTimeEnd"]);
    }
    convertValues(a, classs, asMap = false) {
        if (!a) {
            return a;
        }
        if (Array.isArray(a)) {
            return a.map(elem => this.convertValues(elem, classs));
        }
        else if ("object" === typeof a) {
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
export class Chat {
    constructor(source = {}) {
        Object.defineProperty(this, "id", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "messages", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "title", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "created_at", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "browser_state", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.id = source["id"];
        this.messages = this.convertValues(source["messages"], Message);
        this.title = source["title"];
        this.created_at = this.convertValues(source["created_at"], Time);
        this.browser_state = source["browser_state"];
    }
    convertValues(a, classs, asMap = false) {
        if (!a) {
            return a;
        }
        if (Array.isArray(a)) {
            return a.map(elem => this.convertValues(elem, classs));
        }
        else if ("object" === typeof a) {
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
export class ChatResponse {
    constructor(source = {}) {
        Object.defineProperty(this, "chat", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.chat = this.convertValues(source["chat"], Chat);
    }
    convertValues(a, classs, asMap = false) {
        if (!a) {
            return a;
        }
        if (Array.isArray(a)) {
            return a.map(elem => this.convertValues(elem, classs));
        }
        else if ("object" === typeof a) {
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
export class Model {
    constructor(source = {}) {
        Object.defineProperty(this, "model", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "digest", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "modified_at", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.model = source["model"];
        this.digest = source["digest"];
        this.modified_at = this.convertValues(source["modified_at"], Time);
    }
    convertValues(a, classs, asMap = false) {
        if (!a) {
            return a;
        }
        if (Array.isArray(a)) {
            return a.map(elem => this.convertValues(elem, classs));
        }
        else if ("object" === typeof a) {
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
export class ModelsResponse {
    constructor(source = {}) {
        Object.defineProperty(this, "models", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.models = this.convertValues(source["models"], Model);
    }
    convertValues(a, classs, asMap = false) {
        if (!a) {
            return a;
        }
        if (Array.isArray(a)) {
            return a.map(elem => this.convertValues(elem, classs));
        }
        else if ("object" === typeof a) {
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
export class InferenceCompute {
    constructor(source = {}) {
        Object.defineProperty(this, "library", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "variant", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "compute", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "driver", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "name", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "vram", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.library = source["library"];
        this.variant = source["variant"];
        this.compute = source["compute"];
        this.driver = source["driver"];
        this.name = source["name"];
        this.vram = source["vram"];
    }
}
export class InferenceComputeResponse {
    constructor(source = {}) {
        Object.defineProperty(this, "inferenceComputes", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "defaultContextLength", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.inferenceComputes = this.convertValues(source["inferenceComputes"], InferenceCompute);
        this.defaultContextLength = source["defaultContextLength"];
    }
    convertValues(a, classs, asMap = false) {
        if (!a) {
            return a;
        }
        if (Array.isArray(a)) {
            return a.map(elem => this.convertValues(elem, classs));
        }
        else if ("object" === typeof a) {
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
export class ModelCapabilitiesResponse {
    constructor(source = {}) {
        Object.defineProperty(this, "capabilities", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.capabilities = source["capabilities"];
    }
}
export class ChatEvent {
    constructor(source = {}) {
        Object.defineProperty(this, "eventName", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "content", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "thinking", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "thinkingTimeStart", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "thinkingTimeEnd", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "toolCalls", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "toolCall", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "toolName", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "toolResult", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "toolResultData", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "chatId", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "approvalId", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "approvalScope", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "approvalArgs", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "toolState", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.eventName = source["eventName"];
        this.content = source["content"];
        this.thinking = source["thinking"];
        this.thinkingTimeStart = source["thinkingTimeStart"] && new Date(source["thinkingTimeStart"]);
        this.thinkingTimeEnd = source["thinkingTimeEnd"] && new Date(source["thinkingTimeEnd"]);
        this.toolCalls = this.convertValues(source["toolCalls"], ToolCall);
        this.toolCall = this.convertValues(source["toolCall"], ToolCall);
        this.toolName = source["toolName"];
        this.toolResult = source["toolResult"];
        this.toolResultData = source["toolResultData"];
        this.chatId = source["chatId"];
        this.approvalId = source["approvalId"];
        this.approvalScope = source["approvalScope"];
        this.approvalArgs = source["approvalArgs"];
        this.toolState = source["toolState"];
    }
    convertValues(a, classs, asMap = false) {
        if (!a) {
            return a;
        }
        if (Array.isArray(a)) {
            return a.map(elem => this.convertValues(elem, classs));
        }
        else if ("object" === typeof a) {
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
export class DownloadEvent {
    constructor(source = {}) {
        Object.defineProperty(this, "eventName", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "total", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "completed", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "done", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.eventName = source["eventName"];
        this.total = source["total"];
        this.completed = source["completed"];
        this.done = source["done"];
    }
}
export class ErrorEvent {
    constructor(source = {}) {
        Object.defineProperty(this, "eventName", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "error", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "code", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "details", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.eventName = source["eventName"];
        this.error = source["error"];
        this.code = source["code"];
        this.details = source["details"];
    }
}
export class Settings {
    constructor(source = {}) {
        Object.defineProperty(this, "Expose", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "Browser", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "Survey", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "Models", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "Agent", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "Tools", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "WorkingDir", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "ContextLength", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "TurboEnabled", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "WebSearchEnabled", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "ThinkEnabled", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "ThinkLevel", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "SelectedModel", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "SidebarOpen", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "LastHomeView", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "AutoUpdateEnabled", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "AutoApproveTools", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.Expose = source["Expose"];
        this.Browser = source["Browser"];
        this.Survey = source["Survey"];
        this.Models = source["Models"];
        this.Agent = source["Agent"];
        this.Tools = source["Tools"];
        this.WorkingDir = source["WorkingDir"];
        this.ContextLength = source["ContextLength"];
        this.TurboEnabled = source["TurboEnabled"];
        this.WebSearchEnabled = source["WebSearchEnabled"];
        this.ThinkEnabled = source["ThinkEnabled"];
        this.ThinkLevel = source["ThinkLevel"];
        this.SelectedModel = source["SelectedModel"];
        this.SidebarOpen = source["SidebarOpen"];
        this.LastHomeView = source["LastHomeView"];
        this.AutoUpdateEnabled = source["AutoUpdateEnabled"];
        this.AutoApproveTools = source["AutoApproveTools"];
    }
}
export class SettingsResponse {
    constructor(source = {}) {
        Object.defineProperty(this, "settings", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.settings = this.convertValues(source["settings"], Settings);
    }
    convertValues(a, classs, asMap = false) {
        if (!a) {
            return a;
        }
        if (Array.isArray(a)) {
            return a.map(elem => this.convertValues(elem, classs));
        }
        else if ("object" === typeof a) {
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
export class HealthResponse {
    constructor(source = {}) {
        Object.defineProperty(this, "healthy", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.healthy = source["healthy"];
    }
}
export class User {
    constructor(source = {}) {
        Object.defineProperty(this, "id", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "email", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "name", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "bio", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "avatarurl", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "firstname", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "lastname", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "plan", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.id = source["id"];
        this.email = source["email"];
        this.name = source["name"];
        this.bio = source["bio"];
        this.avatarurl = source["avatarurl"];
        this.firstname = source["firstname"];
        this.lastname = source["lastname"];
        this.plan = source["plan"];
    }
}
export class Attachment {
    constructor(source = {}) {
        Object.defineProperty(this, "filename", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "data", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.filename = source["filename"];
        this.data = source["data"];
    }
}
export class ChatRequest {
    constructor(source = {}) {
        Object.defineProperty(this, "model", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "prompt", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "index", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "attachments", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "web_search", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "file_tools", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "forceUpdate", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "think", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.model = source["model"];
        this.prompt = source["prompt"];
        this.index = source["index"];
        this.attachments = this.convertValues(source["attachments"], Attachment);
        this.web_search = source["web_search"];
        this.file_tools = source["file_tools"];
        this.forceUpdate = source["forceUpdate"];
        this.think = source["think"];
    }
    convertValues(a, classs, asMap = false) {
        if (!a) {
            return a;
        }
        if (Array.isArray(a)) {
            return a.map(elem => this.convertValues(elem, classs));
        }
        else if ("object" === typeof a) {
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
export class Error {
    constructor(source = {}) {
        Object.defineProperty(this, "error", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.error = source["error"];
    }
}
export class ModelUpstreamResponse {
    constructor(source = {}) {
        Object.defineProperty(this, "stale", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "error", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.stale = source["stale"];
        this.error = source["error"];
    }
}
export class Page {
    constructor(source = {}) {
        Object.defineProperty(this, "url", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "title", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "text", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "lines", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "links", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "fetched_at", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.url = source["url"];
        this.title = source["title"];
        this.text = source["text"];
        this.lines = source["lines"];
        this.links = source["links"];
        this.fetched_at = this.convertValues(source["fetched_at"], Time);
    }
    convertValues(a, classs, asMap = false) {
        if (!a) {
            return a;
        }
        if (Array.isArray(a)) {
            return a.map(elem => this.convertValues(elem, classs));
        }
        else if ("object" === typeof a) {
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
export class BrowserStateData {
    constructor(source = {}) {
        Object.defineProperty(this, "page_stack", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "view_tokens", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "url_to_page", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.page_stack = source["page_stack"];
        this.view_tokens = source["view_tokens"];
        this.url_to_page = source["url_to_page"];
    }
}
export class MCPTool {
    constructor(source = {}) {
        Object.defineProperty(this, "name", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "description", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.name = source["name"];
        this.description = source["description"];
    }
}
export class MCPSkippedTool {
    constructor(source = {}) {
        Object.defineProperty(this, "name", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "reason", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.name = source["name"];
        this.reason = source["reason"];
    }
}
export class MCPServer {
    constructor(source = {}) {
        Object.defineProperty(this, "name", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "status", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "transport", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "runs", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "enabled", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "approved", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "changed", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "previouslyRan", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "error", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "tools", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "skipped", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "canSignIn", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "signedIn", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "signingIn", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "warnings", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "tokenStore", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.name = source["name"];
        this.status = source["status"];
        this.transport = source["transport"];
        this.runs = source["runs"];
        this.enabled = source["enabled"];
        this.approved = source["approved"];
        this.changed = source["changed"];
        this.previouslyRan = source["previouslyRan"];
        this.error = source["error"];
        this.tools = this.convertValues(source["tools"], MCPTool);
        this.skipped = this.convertValues(source["skipped"], MCPSkippedTool);
        this.canSignIn = source["canSignIn"];
        this.signedIn = source["signedIn"];
        this.signingIn = source["signingIn"];
        this.warnings = source["warnings"];
        this.tokenStore = source["tokenStore"];
    }
    convertValues(a, classs, asMap = false) {
        if (!a) {
            return a;
        }
        if (Array.isArray(a)) {
            return a.map(elem => this.convertValues(elem, classs));
        }
        else if ("object" === typeof a) {
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
export class MCPServersResponse {
    constructor(source = {}) {
        Object.defineProperty(this, "servers", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.servers = this.convertValues(source["servers"], MCPServer);
    }
    convertValues(a, classs, asMap = false) {
        if (!a) {
            return a;
        }
        if (Array.isArray(a)) {
            return a.map(elem => this.convertValues(elem, classs));
        }
        else if ("object" === typeof a) {
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
export class MCPRegistryEntry {
    constructor(source = {}) {
        Object.defineProperty(this, "name", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "title", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "description", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "version", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "publisher", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "repository", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "websiteUrl", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "installable", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "reason", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "transport", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "runs", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "suggestedName", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "command", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "args", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "env", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "url", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "headers", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "variables", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.name = source["name"];
        this.title = source["title"];
        this.description = source["description"];
        this.version = source["version"];
        this.publisher = source["publisher"];
        this.repository = source["repository"];
        this.websiteUrl = source["websiteUrl"];
        this.installable = source["installable"];
        this.reason = source["reason"];
        this.transport = source["transport"];
        this.runs = source["runs"];
        this.suggestedName = source["suggestedName"];
        this.command = source["command"];
        this.args = source["args"];
        this.env = source["env"];
        this.url = source["url"];
        this.headers = source["headers"];
        this.variables = source["variables"];
    }
}
export class MCPRegistryResponse {
    constructor(source = {}) {
        Object.defineProperty(this, "entries", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "nextCursor", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "notVetted", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.entries = this.convertValues(source["entries"], MCPRegistryEntry);
        this.nextCursor = source["nextCursor"];
        this.notVetted = source["notVetted"];
    }
    convertValues(a, classs, asMap = false) {
        if (!a) {
            return a;
        }
        if (Array.isArray(a)) {
            return a.map(elem => this.convertValues(elem, classs));
        }
        else if ("object" === typeof a) {
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
export class MCPDiscoveredServer {
    constructor(source = {}) {
        Object.defineProperty(this, "name", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "sources", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "paths", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "runs", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "origin", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "notes", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "problem", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "command", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "args", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "env", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "url", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "headers", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "disabled", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.name = source["name"];
        this.sources = source["sources"];
        this.paths = source["paths"];
        this.runs = source["runs"];
        this.origin = source["origin"];
        this.notes = source["notes"];
        this.problem = source["problem"];
        this.command = source["command"];
        this.args = source["args"];
        this.env = source["env"];
        this.url = source["url"];
        this.headers = source["headers"];
        this.disabled = source["disabled"];
    }
}
export class MCPDiscoveryResponse {
    constructor(source = {}) {
        Object.defineProperty(this, "servers", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "searched", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "error", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        if ('string' === typeof source)
            source = JSON.parse(source);
        this.servers = this.convertValues(source["servers"], MCPDiscoveredServer);
        this.searched = source["searched"];
        this.error = source["error"];
    }
    convertValues(a, classs, asMap = false) {
        if (!a) {
            return a;
        }
        if (Array.isArray(a)) {
            return a.map(elem => this.convertValues(elem, classs));
        }
        else if ("object" === typeof a) {
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
