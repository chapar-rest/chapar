package org.gioui.x.explorer;

import java.io.InputStream;
import java.io.OutputStream;
import java.io.Closeable;
import java.io.Flushable;

public class file_android {
    public String err;
    public Object handler;

    public void setHandle(Object f) {
        this.handler = f;
    }

    public int fileRead(byte[] b) {
        try {
            return ((InputStream) this.handler).read(b, 0, b.length);
        } catch (Exception e) {
            this.err = e.toString();
            return 0;
        }
    }

    public boolean fileWrite(byte[] b) {
        try {
            ((OutputStream) this.handler).write(b);
            return true;
        } catch (Exception e) {
            this.err = e.toString();
            return false;
        }
    }

    public boolean fileClose() {
        try {
            if (this.handler instanceof Flushable) {
                ((Flushable) this.handler).flush();
            }
            if (this.handler instanceof Closeable) {
                ((Closeable) this.handler).close();
            }
            return true;
        } catch (Exception e) {
            this.err = e.toString();
            return false;
        }
    }

    public String getError() {
        return this.err;
    }

}