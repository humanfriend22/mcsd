import { createDiscreteApi, darkTheme } from 'naive-ui'

const { message } = createDiscreteApi(['message'], { configProviderProps: { theme: darkTheme } })

export function handleError(msg: string, status?: number) {
    const prefix = status && status >= 400 && status < 500 ? 'Client: ' : 'Server: '
    message.error(prefix + msg);
    console.error(prefix + msg);
};

export async function handleRequestError(response: Response) {
    if (!response.ok) {
        let msg = response.statusText
        try {
            const body = await response.json()
            if (body.error) msg = body.error
        } catch {}
        throw { message: msg, status: response.status }
    }
};